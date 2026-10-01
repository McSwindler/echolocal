package rtspd

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/ygelfand/echolocal/internal/android/prop"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/vision"
	"github.com/ygelfand/echolocal/internal/layout"
	"github.com/ygelfand/echolocal/internal/lib/onvif"
)

const (
	ONVIFPort    = 8000
	snapshotPath = "/onvif/snapshot.jpg"
)

func snapshot(w http.ResponseWriter, _ *http.Request) {
	jpeg, err := vision.Get().Still()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(jpeg)
}

type describer struct {
	http  *http.Server
	found *onvif.Responder
}

func profiles() []onvif.Profile { return helperProfiles() }

func describe() *describer {
	dev := config.Get().Device
	serial, _ := prop.Get("ro.serialno")
	if serial == "" {
		serial = dev.Name
	}
	model := dev.Board.Model
	svc := &onvif.Service{Device: onvif.Device{
		Manufacturer: layout.Manufacturer, Model: model, Firmware: layout.Version, Serial: serial, Hardware: model,
		Name: dev.Name, RTSPPort: Port, Profiles: profiles, SnapshotPath: snapshotPath,
	}}
	mux := http.NewServeMux()
	mux.Handle(onvif.DevicePath, svc)
	mux.Handle(onvif.MediaPath, svc)
	mux.HandleFunc(snapshotPath, snapshot)

	d := &describer{http: &http.Server{Handler: mux}}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(ONVIFPort))
	if err != nil {
		slog.Warn("onvif is not answering", "port", ONVIFPort, "err", err)
		return d
	}
	go d.http.Serve(ln)

	d.found = &onvif.Responder{UUID: onvif.NewUUID(), Name: dev.Name, XAddr: func(ip net.IP) string {
		return "http://" + net.JoinHostPort(ip.String(), strconv.Itoa(ONVIFPort)) + onvif.DevicePath
	}}
	if err := d.found.Listen(); err != nil {
		slog.Warn("onvif discovery is not answering", "err", err)
		d.found = nil
	}
	return d
}

func (d *describer) close() {
	if d == nil {
		return
	}
	d.http.Close()
	if d.found != nil {
		d.found.Close()
	}
}
