// Package layout is the on-device layout echoctl writes and echod reads. Both binaries have to
// agree on every name here, so they come from one place rather than being repeated.
package layout

import (
	"fmt"
	"os"
	"strings"
)

// Where echod and its state live. /system is read-only once installed, so anything written
// after install goes on /data.
const (
	Dir      = "/system/app/echod"
	Binary   = Dir + "/echod"
	Surface  = Dir + "/echolocal-surface"
	Camera   = Dir + "/echolocal-camera"
	CamShim  = Dir + "/libecholocal-camshim.so"
	StateDir = "/data/misc/echolocal"
	KeyPath  = StateDir + "/psk"
	NamePath = StateDir + "/name"

	CastCredentialsPath = StateDir + "/cast-credentials.json"
	CastAuthorityPath   = StateDir + "/cast-authority.json"

	// Service is echod's own init service, on a board that has one rather than taking over Amazon's.
	FBDevice = "/dev/graphics/fb0"

	Service        = "echod"
	SurfaceService = "echolocal_sf"
	InitRC         = "/system/etc/init/echolocal.rc"

	CameraService       = "echolocal_cam"
	CameraServerService = "cameraserver"
	CameraSocket        = "/dev/socket/echolocal-camera"

	// PrevBinary is the binary an update replaced, kept until the new one has proved itself. Its
	// presence at boot is what says a trial never finished, so nothing may leave one lying around.
	// OldBinary is where a proven update files it, one generation back.
	PrevBinary = StateDir + "/echod.prev"
	OldBinary  = StateDir + "/echod.old"

	// Where older releases keep them, on /system beside the binary.
	SystemPrevBinary = Binary + ".prev"
	SystemOldBinary  = Binary + ".old"

	// AsideBinary holds the running binary on /system while its replacement is written.
	AsideBinary = Binary + ".aside"

	// UpdatingPath holds the version being tried, so a rollback can say which one it took out. It is
	// under /data because the boot hook reads it after a restore has already remounted /system back to
	// read-only.
	UpdatingPath = StateDir + "/updating"
)

// The boot animation wrappers init runs; both call ledctrl, which waits on a binder service echod does
// not publish. StartAnimation runs on the way up and StopAnimation once Android reports the boot
// finished, which is the difference that decides what may go in them.
const (
	StartAnimation = "/system/bin/start_animation.sh"
	StopAnimation  = "/system/bin/stop_animation.sh"
)

// OurLabel is the SELinux label echod's own files carry. A service's domain comes from the label of
// the file init execs, and system_file has no transition rule, which leaves echod in init's own
// domain. The label the stock binary wore is the board's, since it names that board's service.
const OurLabel = "u:object_r:system_file:s0"

// Properties echod publishes about itself. Started is an uptime, which only moves forward
// within a boot, so a changed value means a new process.
const (
	StartedProp = "echolocal.started"
	StateProp   = "echolocal.state"

	// TrialProp marks that a process this boot already took an update and has not committed it.
	// Deliberately not a persist property: it has to survive init restarting echod, which is what
	// makes a second attempt recognisable, and it has to be forgotten across a reboot, which is what
	// gives the boot hook its turn.
	TrialProp = "echolocal.trial"

	// RolledBackProp is set by the boot hook when it puts the previous binary back, so the failure
	// reaches Home Assistant instead of only logcat.
	RolledBackProp = "echolocal.rolledback"
)

// Where the supplicant keeps its configuration and control socket. echoctl writes the configuration
// at install and echod drives the supplicant over the socket, so both have to name the same paths.
const (
	WifiDir     = "/data/misc/wifi"
	WifiConf    = WifiDir + "/wpa_supplicant.conf"
	WifiSockets = WifiDir + "/sockets"
	WifiIface   = "wlan0"
	WifiUser    = 1010
)

// LogTag is echod's logcat tag: `adb logcat -s echolocal`.
const LogTag = "echolocal"

// Port is the ESPHome native API port Home Assistant expects.
const Port = 6053

// FirewallHook is a script Amazon's firewall.sh runs if it exists, after building its allowlist
// and after flushing INPUT. Occupying it is how our port stays open across every invocation
// without editing their script. Greengrass is not installed and nothing else references it.
const FirewallHook = "/system/bin/greengrass_firewall.sh"

// MaxNodeName is the length limit the ESPHome API imposes on a node name.
const MaxNodeName = 31

// Hardware identity, as Home Assistant shows it in the device panel. What varies between models —
// the model name, the board and the fallback display name — belongs to the board.
const (
	Manufacturer = "EchoLocal"
	Platform     = "echolocal"
)

// StatePath holds echod's runtime settings.
const StatePath = StateDir + "/state.json"

// Wake word models live in /data, not /system: Home Assistant can offer new ones at runtime and
// /system is mounted read-only.
const ModelDir = StateDir + "/models"

// RecordingDir holds the kept turn audio, one WAV and one metadata file per turn, named by turn id.
const RecordingDir = StateDir + "/recordings"

const TempDir = StateDir + "/tmp"

// MAC normalizes an address into the form Home Assistant compares against, and reports "" for
// anything that would not identify a device. idme writes twelve hex digits with no separators.
func MAC(raw string) string {
	var digits strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'f' {
			digits.WriteRune(r)
		}
	}

	s := digits.String()
	if len(s) != 12 || s == "000000000000" {
		return ""
	}

	var mac strings.Builder
	for i := 0; i < len(s); i += 2 {
		if i > 0 {
			mac.WriteByte(':')
		}
		mac.WriteString(s[i : i+2])
	}
	return mac.String()
}

// FactoryMAC reads and normalizes the immutable address recorded for the Wi-Fi device.
func FactoryMAC() (string, error) {
	raw, err := os.ReadFile(MACPath)
	if err != nil {
		return "", fmt.Errorf("reading the factory MAC: %w", err)
	}
	mac := MAC(string(raw))
	if mac == "" {
		return "", fmt.Errorf("%s holds %q, which is not an address", MACPath, strings.TrimSpace(string(raw)))
	}
	return mac, nil
}

// Idme reads a factory identity field from /proc/idme, empty on any error. These are written at
// manufacture and never change, so a caller reads once and holds it.
func Idme(name string) string {
	raw, err := os.ReadFile("/proc/idme/" + name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// Slug turns a display name into a node name: an mDNS hostname, and the prefix Home Assistant
// builds entity ids from. "Living Room" becomes "living-room".
func Slug(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
