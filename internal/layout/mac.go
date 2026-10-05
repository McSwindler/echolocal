//go:build !board_doppler

package layout

// MACPath is the address the factory recorded, which the Wi-Fi driver takes when it comes up. idme
// is a kernel interface, so it reads this early in boot, before wlan0 exists and without /data.
const MACPath = "/proc/idme/mac_addr"
