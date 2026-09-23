.DEFAULT_GOAL := help

GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

BASE := $(shell cat VERSION 2>/dev/null || echo 0.0.0)
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null)
EPOCH := $(shell date -u +%s)
BUILT_FROM := $(shell git status --porcelain 2>/dev/null | grep -q . && echo dirty || echo $(GIT_COMMIT))
VERSION ?= $(if $(GIT_TAG),$(GIT_TAG),$(BASE)-dev.$(EPOCH)_$(BUILT_FROM))

REPO ?= ygelfand/echolocal
RELEASES ?= https://github.com/$(REPO)/releases

BUILDVARS := github.com/ygelfand/echolocal/internal/layout
BOARDVARS := github.com/ygelfand/echolocal/internal/board
LDFLAGS := -X '$(BUILDVARS).Version=$(VERSION)' \
	-X '$(BUILDVARS).GitCommit=$(GIT_COMMIT)' \
	-X '$(BUILDVARS).BuildDate=$(BUILD_DATE)'

BUILD_DIR := bin
ASSET_DIR := internal/host/assets/payload

# echod targets the Echo Dot 2: MT8163, Android 5.1 (API 22). FireOS 5 runs an arm64 kernel; FireOS 6
# ships a 32-bit kernel on the same hardware, which cannot exec arm64 at all.
# The ALSA path is pure Go over /dev/snd ioctls, so no cgo and no NDK. Keep it that way unless
# something genuinely needs C, which is what would make a toolchain image worth having.
ARCHES := arm64 arm
DOT_ARCH ?= arm
DEVICE_ENV := GOOS=linux GOARCH=$(DOT_ARCH) CGO_ENABLED=0

# BOARD builds for one board: links its board_<codename> packages and pins Detect. Unset builds for all.
BOARD ?=

# Boards needing a build of their own, read off the board tags in component/all. Empty is the normal
# case, where every board takes the shared binary.
BOARD_BUILDS := $(shell grep -h '^//go:build' internal/component/all/*.go 2>/dev/null | \
	grep -o 'board_[a-z0-9]*' | sed 's/^board_//' | sort -u)

# TAGS passes build tags through to echod, which is how the same device can be measured both ways:
# `make install-echod TAGS=noasm` builds the portable dot instead of the NEON one.
TAGS ?=
empty :=
space := $(empty) $(empty)
comma := ,
ALL_TAGS := $(strip $(TAGS) $(if $(BOARD),board_$(BOARD)))
DEVICE_TAGS := $(if $(ALL_TAGS),-tags $(subst $(space),$(comma),$(ALL_TAGS)),)

DEVICE_LDFLAGS := -s -w $(LDFLAGS) $(if $(BOARD),-X '$(BOARDVARS).pinned=$(BOARD)')
DEVICE_BIN := $(BUILD_DIR)/echod$(if $(BOARD),-$(BOARD))-$(DOT_ARCH)

ADB ?= adb
DEVICE_TMP := /data/local/tmp

# DEVICE names which attached device the device targets act on, when there is more than one.
DEVICE ?=
ifneq ($(DEVICE),)
ADB := $(ADB) -s $(DEVICE)
SERIAL := --serial $(DEVICE)
endif

# What the attached device is, as shell assignments: codename, device and service. echoctl reads the
# board table, so this is not a second copy of it.
BOARD_SH = go run ./cmd/echoctl board --sh $(SERIAL)

# The device targets build with BOARD only for a board with packages of its own. Everywhere else the
# plain binary is what the manifest serves, and installing another would be debugging the wrong one.

# echod lives under /system/app because that tree is labelled u:object_r:system_file:s0,
# the label that leaves an init-started service in init's own domain rather than the narrow
# per-service domain its stock *_exec label would select.
#
# It is installed as one of Amazon's own services — which one is the board's, see internal/board —
# so init starts echod from on post-fs-data and restarts it if it exits.
ECHOD_DIR := /system/app/echod
STATE_DIR := /data/misc/echolocal

##@ Development

.PHONY: build
build: build-echoctl build-echod ## Build both binaries

.PHONY: build-echoctl
build-echoctl: ## Build the host CLI into ./bin
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/echoctl ./cmd/echoctl

.PHONY: build-echod
build-echod: ## Cross-compile echod for the Echo Dot (DOT_ARCH=arm64 for a Fire OS 5 kernel; TAGS=noasm for the portable dot)
	@mkdir -p $(BUILD_DIR)
	$(DEVICE_ENV) go build $(DEVICE_TAGS) -ldflags "$(DEVICE_LDFLAGS)" -o $(DEVICE_BIN) ./cmd/echod

.PHONY: build-echod-all
build-echod-all: ## Build every binary a release publishes, from a clean bin
	@rm -f $(BUILD_DIR)/echod-*
	@for a in $(ARCHES); do $(MAKE) --no-print-directory build-echod DOT_ARCH=$$a; done
	@for b in $(BOARD_BUILDS); do \
		for a in $(ARCHES); do $(MAKE) --no-print-directory build-echod DOT_ARCH=$$a BOARD=$$b; done; \
	done

.PHONY: run-echoctl
run-echoctl: ## Run echoctl on the host (make run-echoctl ARGS="tools tone -h")
	go run ./cmd/echoctl $(ARGS)

.PHONY: run-echod
run-echod: push-echod ## Push echod and run it (make run-echod ARGS="tools info")
	$(ADB) shell $(DEVICE_TMP)/echod $(ARGS)

.PHONY: push-echod
push-echod: build-echod ## Push echod to /data/local/tmp for iteration
	@$(ADB) push $(DEVICE_BIN) $(DEVICE_TMP)/echod >/dev/null
	@$(ADB) shell chmod 755 $(DEVICE_TMP)/echod

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: test-race
test-race: ## Run tests with the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run tests and open a coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: fmt
fmt: ## Format Go source
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: check
check: fmt vet lint test ## Format, vet, lint and test

##@ Device (echod)

.PHONY: device
device: ## Say which device the device targets would write to, and refuse an unknown one
	@n=$$($(ADB) devices | grep -c "device$$"); \
	if [ "$$n" -eq 0 ]; then echo "no device attached"; exit 1; fi; \
	if [ "$$n" -gt 1 ] && [ -z "$(DEVICE)" ]; then \
		echo "$$n devices attached, so name one:"; \
		$(ADB) devices | grep "device$$" | sed 's/^/  make DEVICE=/;s/\tdevice$$//'; \
		exit 1; \
	fi
	@$(BOARD_SH) >/dev/null || { \
		echo "these targets remount /system, so an unrecognised device is refused"; exit 1; }
	@go run ./cmd/echoctl board $(SERIAL)

.PHONY: payload
payload: ## Stage echod for embedding into echoctl
	@$(MAKE) --no-print-directory build-echod DOT_ARCH=arm
	@mkdir -p $(ASSET_DIR)
	cp $(BUILD_DIR)/echod-arm $(ASSET_DIR)/echod
	@shasum -a 256 $(ASSET_DIR)/echod | awk '{print $$1}' > $(ASSET_DIR)/echod.sha256

.PHONY: dist
dist: payload ## Full build: echod, then echoctl carrying it
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags payload -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/echoctl ./cmd/echoctl

.PHONY: install-echod
install-echod: ## Install echod into /system/app/echod, built for whatever board is attached
# This is a manual copy, not an upgrade, so the trial an update may have left open is cleared with it.
# Otherwise the restart below looks like a binary that took an update and died without committing, and
# echod reboots the device to put the old one back — taking this install with it.
	@$(MAKE) --no-print-directory device
	@eval "$$($(BOARD_SH))"; \
		board=$$(echo "$(BOARD_BUILDS)" | tr ' ' '\n' | grep -x "$$codename" || true); \
		$(MAKE) --no-print-directory build-echod $${board:+BOARD=$$board}; \
		bin=$(BUILD_DIR)/echod$${board:+-$$board}-$(DOT_ARCH); \
		$(ADB) shell "setprop ctl.stop $$service; sleep 1"; \
		$(ADB) remount >/dev/null; \
		$(ADB) shell 'mkdir -p $(ECHOD_DIR) && rm -f $(ECHOD_DIR)/echod.prev $(ECHOD_DIR)/echod.old'; \
		$(ADB) push "$$bin" $(ECHOD_DIR)/echod >/dev/null; \
		$(ADB) shell "chmod 755 $(ECHOD_DIR)/echod; \
			rm -f $(STATE_DIR)/updating; setprop echolocal.trial ''; setprop echolocal.rolledback ''; \
			[ -L /system/bin/$$service ] && setprop ctl.start $$service; ls -lZ $(ECHOD_DIR)/echod"

.PHONY: install-service
install-service: install-echod ## Take over the board's service so init starts echod
	@eval "$$($(BOARD_SH))"; svc=/system/bin/$$service; \
		$(ADB) remount >/dev/null; \
		$(ADB) shell "[ -e $$svc.orig ] || mv $$svc $$svc.orig; \
			rm -f $$svc; ln -s $(ECHOD_DIR)/echod $$svc; ls -lZ $$svc $$svc.orig"

.PHONY: uninstall-service
uninstall-service: ## Restore Amazon's binary for the board's service
	@eval "$$($(BOARD_SH))"; svc=/system/bin/$$service; \
		$(ADB) remount >/dev/null; \
		$(ADB) shell "rm -f $$svc; mv $$svc.orig $$svc; chcon $$label $$svc; ls -lZ $$svc"

.PHONY: restart-echod
restart-echod: ## Restart echod through init (ctl.stop then ctl.start)
	@eval "$$($(BOARD_SH))"; \
		$(ADB) shell "setprop ctl.stop $$service; sleep 1; setprop ctl.start $$service; \
			sleep 1; echo \"init.svc: \$$(getprop init.svc.$$service)\""

.PHONY: state
state: ## Show what echod and init say about echod
	@eval "$$($(BOARD_SH))"; \
		$(ADB) shell "echo \"state:   \$$(getprop echolocal.state)\"; \
			echo \"started: \$$(getprop echolocal.started)\"; \
			echo \"init.svc: \$$(getprop init.svc.$$service)\""

.PHONY: logs
logs: ## Tail echod logs from a connected device
	$(ADB) logcat -s echolocal:*

.PHONY: shell
shell: ## Open a root shell on a connected device
	$(ADB) shell

##@ Build & Release

.PHONY: install
install: ## go install echoctl into $$GOPATH/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/echoctl

AT ?= $(VERSION)
FROM ?= $(RELEASES)/download/$(AT)
PAGE ?= $(RELEASES)/tag/$(AT)

.PHONY: manifest
manifest: build-echod-all ## Write the manifest a device fetches to find this build
	@mkdir -p $(BUILD_DIR)
	go run ./cmd/mkmanifest \
		-version "$(AT)" \
		-from "$(FROM)" \
		-arm64 $(BUILD_DIR)/echod-arm64 \
		-arm $(BUILD_DIR)/echod-arm \
		$(foreach b,$(BOARD_BUILDS),$(foreach a,$(ARCHES),-board $(b):$(a):$(BUILD_DIR)/echod-$(b)-$(a) )) \
		-title "EchoLocal $(AT)" \
		-release-url "$(PAGE)" \
		-out $(BUILD_DIR)/manifest.json
	@cat $(BUILD_DIR)/manifest.json

# Boot images are published once per board, under one tag of their own, and never republished: the
# hashes are compiled into echoctl, so what this tag serves has to keep matching every build that
# has already shipped. That is why they are not goreleaser extra_files, which attach to whichever
# tag is being released.
BOOT_TAG := boot-images
BOOT_DIR := internal/host/assets/boot

LOGO_SRC := assets
LOGO_OUT := internal/ui

.PHONY: logos
logos: ## Rescale the committed artwork into what echod embeds
	@for n in dark light; do \
		go run ./cmd/mklogo -in $(LOGO_SRC)/logo_$$n.png -out $(LOGO_OUT)/logo_$$n.png; \
	done

.PHONY: boot-images
boot-images: ## Publish the boot images echoctl fetches at install time (run once per board)
	@command -v gh >/dev/null || { echo "needs the gh CLI: brew install gh"; exit 1; }
	@go test ./internal/host/bootimg/ -run TestEveryImageIsInTheTreeAndIsWhatWeSayItIs
	@gh release view $(BOOT_TAG) >/dev/null 2>&1 || \
		gh release create $(BOOT_TAG) --title "Boot images" \
			--notes "Per-board boot images, fetched by echoctl at install time against a hash compiled into it. Uploaded once per board and never replaced."
	gh release upload $(BOOT_TAG) $(BOOT_DIR)/*.img --clobber
	@echo "$(BOOT_TAG) now serves the boot images"

.PHONY: release-dev
release-dev: ## Publish this working tree to the dev channel, without pushing anything
	@command -v gh >/dev/null || { echo "needs the gh CLI: brew install gh"; exit 1; }
	@command -v goreleaser >/dev/null || { echo "needs goreleaser: brew install goreleaser"; exit 1; }
	VERSION=$(VERSION) goreleaser release --snapshot --clean
	@for f in dist/echoctl_*/echoctl dist/echoctl_*/echoctl.exe; do \
		[ -f "$$f" ] || continue; \
		d=$$(basename $$(dirname $$f)); \
		os=$$(echo $$d | cut -d_ -f2); \
		arch=$$(echo $$d | cut -d_ -f3); \
		if [ "$$arch" = amd64 ]; then arch=x86_64; fi; \
		ext=$${f##*.}; [ "$$ext" = exe ] && ext=.exe || ext=; \
		cp "$$f" "dist/echolocal_$${os}_$${arch}$${ext}"; \
	done
	@$(MAKE) --no-print-directory manifest VERSION=$(VERSION) FROM=$(RELEASES)/download/dev PAGE=$(RELEASES)/tag/dev
	@gh release view dev >/dev/null 2>&1 || \
		gh release create dev --prerelease --title dev --notes "Rolling build for devices on the dev channel."
	gh release upload dev dist/echolocal_* $(BUILD_DIR)/echod-* $(BUILD_DIR)/manifest.json --clobber
	@echo "dev channel now serves $(VERSION)"

.PHONY: release
release: ## Release the version in VERSION
	@$(MAKE) --no-print-directory tag TAG=$(BASE)

.PHONY: tag
tag:
	@test -n "$(TAG)" || { echo "usage: make release, make release-dev, or make tag TAG=0.4.2"; exit 1; }
	@test -z "$$(git status --porcelain)" || { echo "the working tree is dirty"; exit 1; }
	@git rev-parse -q --verify "refs/tags/$(TAG)" >/dev/null && { echo "$(TAG) already exists"; exit 1; } || true
	git tag -a "$(TAG)" -m "EchoLocal $(TAG)"
	git push origin "$(TAG)"
	@echo "pushed $(TAG) — the release workflow builds it from here"

.PHONY: snapshot
snapshot: ## Build a local goreleaser snapshot
	goreleaser release --snapshot --clean

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) dist/ coverage.out
	rm -rf $(ASSET_DIR)
	go clean

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
