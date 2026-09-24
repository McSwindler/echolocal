//go:build !linux

package clock

import (
	"fmt"
	"time"
)

func setClock(time.Time) error { return fmt.Errorf("clock: setting the clock needs linux") }
