package media

// Controller is the transport the player's buttons reach.
type Controller interface {
	Play()
	Pause()
	Stop()
	Next()
	Previous()
}

// Transport is whoever the buttons reach: whoever holds the card, or this player when nothing does.
func Transport() Controller {
	if c, ok := Get().Holder().(Controller); ok {
		return c
	}
	return queue{}
}

type queue struct{}

func (queue) Play()     {}
func (queue) Pause()    {}
func (queue) Stop()     { Get().Stop() }
func (queue) Next()     { Get().Next() }
func (queue) Previous() { Get().Previous() }
