// Package wifi owns the radio without any of Android's networking: the interface, and the
// supplicant that associates. Addresses belong to the dhcp feature.
package wifi

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ygelfand/echolocal/internal/layout"
)

// Interface is the station the supplicant runs on.
const Interface = layout.WifiIface

// Loaded reports whether the interface exists.
func Loaded() bool {
	_, err := os.Stat("/sys/class/net/" + Interface)
	return err == nil
}

// appear is how long the interface is waited for. init insmods the driver from our boot image, so
// echod may start first.
const appear = 15 * time.Second

// Load waits for the driver init loaded to register the interface.
func Load() error {
	deadline := time.Now().Add(appear)
	for {
		if Loaded() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not appear within %s", Interface, appear)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// MAC is the interface's address, which is the device's identity to Home Assistant.
func MAC() (string, error) {
	b, err := os.ReadFile("/sys/class/net/" + Interface + "/address")
	if err != nil {
		return "", fmt.Errorf("reading the wlan address: %w", err)
	}
	mac := strings.TrimSpace(string(b))
	if mac == "" || mac == "00:00:00:00:00:00" {
		return "", fmt.Errorf("the wlan address reads %q", mac)
	}
	return mac, nil
}
