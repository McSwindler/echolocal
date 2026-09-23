package indicate

import (
	"sync"
	"time"
)

var (
	mu         sync.Mutex
	registered []func() Surface
)

// Register adds a surface, in preference order. make returns nil where the board lacks it. It exists
// so this package does not import the hardware it stands in front of.
func Register(make func() Surface) {
	mu.Lock()
	defer mu.Unlock()
	registered = append(registered, make)
}

var (
	once  sync.Once
	found Surface
)

// Get is what this board shows things on, and a surface that shows nothing where it has none. It
// always answers, so a feature says what happened without asking whether anyone can see it.
func Get() Surface {
	once.Do(func() {
		mu.Lock()
		defer mu.Unlock()

		for _, make := range registered {
			if s := make(); s != nil && s.Present() {
				found = s
				return
			}
		}
		found = nowhere{}
	})
	return found
}

type nowhere struct{}

func (nowhere) Claim(Priority) Claim { return dropped{} }
func (nowhere) HoldOnStop(Color)     {}
func (nowhere) Present() bool        { return false }

type dropped struct{}

func (dropped) Play(string, Color)                   {}
func (dropped) PlayReversed(string, Color)           {}
func (dropped) React(string, Color, Room)            {}
func (dropped) ShowFor(string, Color, time.Duration) {}
func (dropped) Clear()                               {}
func (dropped) Release()                             {}
