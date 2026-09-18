package boot

import (
	"fmt"
	"os"
	"strings"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/layout"
)

func listenAddr() string { return fmt.Sprintf(":%d", layout.Port) }

func name(b board.Board) string {
	if raw, err := os.ReadFile(layout.NamePath); err == nil {
		if recorded := strings.TrimSpace(string(raw)); recorded != "" {
			return recorded
		}
	}

	mac, err := layout.FactoryMAC()
	if err != nil {
		return b.DefaultName
	}
	return b.NameFromMAC(mac)
}
