package installer

import (
	"github.com/ygelfand/echolocal/internal/android/wifi"
	"github.com/ygelfand/echolocal/internal/board"
)

// bringUpWifi leaves the radio answering, so `echoctl wifi` has a supplicant to talk to.
func bringUpWifi(r *run) (string, bool, error) {
	if !r.board().Has(board.Wifi) {
		return "the platform brings the radio up", true, nil
	}

	detail, err := wifi.Up(r.d)
	if err != nil {
		return "", false, err
	}
	return detail, detail == "already up", nil
}
