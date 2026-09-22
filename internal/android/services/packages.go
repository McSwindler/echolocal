package services

// Package is one Amazon package an install hides, and why it goes.
type Package struct {
	Name   string
	Reason string
}

// Hidden is what any board with a package manager hides, which is not biscuit: Fire OS 6 there has no
// framework and nothing to hide.
//
// `pm hide` persists, so this is applied once at install rather than on every boot. It is paired with
// a force-stop, because hiding a package blocks future launches but leaves a running process alive,
// and a live audio client keeps mediaserver holding the PCM devices. A package marked persistent is
// restarted by system_server whatever the force-stop does, and only goes on the next boot.
//
// The list was established by hand on a Fire OS 5 biscuit; these are the entries a Show still has
// under the same name. A board adds its own by appending.
var Hidden = []Package{
	{"amazon.speech.sim", "owns the mute button and mic mute state; persistent, so it goes on the next boot"},
	{"amazon.speech.davs.davcservice", "Alexa voice service; holds capture"},
	{"com.amazon.device.echoaudioservice", "audio service; holds the PCM devices"},
	{"com.amazon.alexa.externalmediaplayer.fireos", "playback agent"},
	{"com.amazon.alexa.beaconbroadcaster", "Alexa beacon broadcast"},
	{"com.amazon.spotify.mediabrowserservice", "media browser"},
	{"com.amazon.wha.mediabrowserservice", "media browser"},
	{"com.amazon.whad", "whole-home audio; advertises a control plane on the LAN"},
	{"com.amazon.NativeAccessorProxyServices", "Alexa SIM directive and OOBE receivers"},

	{"com.amazon.device.smarthome.dshs.services", "smart home service"},
	{"com.amazon.device.smarthome.dshs.endpointdetectorCA", "smart home endpoint detection"},
	{"com.amazon.device.smarthome.adapters.ble", "smart home BLE adapter"},
	{"com.amazon.device.smarthome.adapters.echo", "smart home Echo adapter"},
	{"com.amazon.device.smarthome.adapters.wifi", "smart home wifi adapter"},
	{"com.amazon.device.gadgetscontrolmanager", "Echo Buttons and other gadgets"},

	{"com.amazon.android.service.wifiprofilemanager", "re-asserts its own network through WifiManager, leaving no trace on disk"},
	{"com.amazon.whisperjoin.wss.wifiprovisioner", "Wi-Fi Simple Setup provisioner"},

	{"com.amazon.device.smarthome.ota", "smart home OTA"},
	{"com.amazon.device.software.ota", "firmware OTA"},

	{"com.amazon.banyan.core", "metrics pipeline"},
	{"com.amazon.diode", "analytics, and analysis of connectivity events"},
	{"com.amazon.device.logmanager", "log collection"},
	{"com.amazon.device.crashmanager", "uploads crash dumps"},
	{"com.amazon.imp", "Amazon account and OAuth"},
	{"com.amazon.adep", "remote device management and Arcus remote config"},
	{"com.amazon.tcomm", "Amazon's cloud transport"},
	{"com.amazon.device.messaging", "push messaging from Amazon"},
	{"com.amazon.kindleautomatictimezone", "timezone from an Amazon account, woken by joining wifi"},
	{"com.amazon.device.settings", "the settings app"},
	{"com.android.bluetooth", "Android's Bluetooth stack; holds /dev/stpbt"},
}
