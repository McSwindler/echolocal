package visuals

import "github.com/ygelfand/echolocal/internal/hardware/mic"

// listenMono is the microphone mix as stereo, the same frame on both sides.
func listenMono() (<-chan []int16, func()) {
	in, stop := mic.Get().ListenUnleveled("visuals")
	out := make(chan []int16, cap(in))
	go func() {
		defer close(out)
		for f := range in {
			s := make([]int16, 2*len(f))
			for i, v := range f {
				s[2*i], s[2*i+1] = v, v
			}
			out <- s
		}
	}()
	return out, stop
}
