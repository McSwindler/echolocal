package mic

// Echo is one frame at three points: the center microphone as captured, the playback loopback, and
// what listeners were handed.
type Echo struct {
	Mic, Ref, Out []int16
	Sounding      bool
}

// ListenEcho returns every frame at all three points, for measuring the canceller.
func (s *Source) ListenEcho() (<-chan Echo, func()) {
	ch := make(chan Echo, 64)

	s.mu.Lock()
	id := s.next
	s.next++
	s.echoes[id] = ch
	s.mu.Unlock()

	return ch, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, ok := s.echoes[id]; ok {
			delete(s.echoes, id)
			close(c)
		}
	}
}
