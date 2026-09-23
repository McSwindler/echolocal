package component

import (
	"cmp"
	"context"
	"slices"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/board"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/service"
)

// Registry is everything registered, in the order it happens.
//
// Order is declared, never inferred. Components register from init, and Go decides that order from
// the import graph — which is not something the ring coming up before the splash should depend on.
// So entries sort by phase, then by a declared Order, then by name, and the result is the same
// whatever the linker did.
type Registry struct {
	mu      sync.Mutex
	entries []*entry

	// board is what the components are being built for, and told whether anything has said.
	board board.Board
	told  bool
}

type entry struct {
	make  func() Component
	once  sync.Once
	c     Component
	phase Phase
	order int
	opts  []service.Option

	// needs is the hardware this component has nothing to do without.
	needs board.Cap
}

// resolve builds the component, once. Every walk goes through sorted, which resolves before anything
// reads c.
func (e *entry) resolve() { e.once.Do(func() { e.c = e.make() }) }

// Option adjusts one registration.
type Option func(*entry)

// Order places a component within its phase. Lower comes first; equal orders fall back to the name.
func Order(n int) Option { return func(e *entry) { e.order = n } }

// Supervise passes the policy through to the supervisor, for a component with a loop.
func Supervise(opts ...service.Option) Option {
	return func(e *entry) { e.opts = append(e.opts, opts...) }
}

// Needs says the component has nothing to do on a board without this hardware. It is left out
// entirely there: never built, so it never opens a device, and with no entity, action or lifecycle
// of its own. Home Assistant is shown a device that does not have the thing, rather than one whose
// controls do nothing.
func Needs(c board.Cap) Option { return func(e *entry) { e.needs |= c } }

// New is an empty registry. There is a process-wide one for components to register into; this is
// for tests, which want their own.
func New() *Registry { return &Registry{} }

// Use says which board the components are being built for. It has to be called before anything
// walks the registry, since the first walk is what builds them.
func (r *Registry) Use(b board.Board) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.board, r.told = b, true
}

// Board is which device this is, asking the device itself when nobody has said. The offline tools
// never call Use.
func (r *Registry) Board() board.Board {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.onBoard()
}

// onBoard is Board with the lock already held.
func (r *Registry) onBoard() board.Board {
	if !r.told {
		r.board, r.told = board.Detect(), true
	}
	return r.board
}

// Board is the process-wide answer to which device this is.
//
// Components ask here rather than detecting for themselves, so there is one answer and not several
// that happen to agree: Needs is resolved against this, and a component deciding for itself whether
// to offer a setting has to reach the same conclusion the registry did.
func Board() board.Board { return shared.Board() }

var shared = New()

// Register adds to the process-wide registry, from a component package's init.
//
// What is registered is the constructor, not the component: init runs on every invocation of the
// binary, and building a component opens the hardware it drives. Only the agent should do that.
func Register[T Component](p Phase, make func() T, opts ...Option) {
	shared.Add(p, func() Component { return make() }, opts...)
}

// Default is the process-wide registry.
func Default() *Registry { return shared }

// Add registers a component, built on first use and once.
func (r *Registry) Add(p Phase, make func() Component, opts ...Option) {
	e := &entry{make: make, phase: p}
	for _, o := range opts {
		o(e)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
}

// All is every component, in the order they come up.
func (r *Registry) All() []Component {
	out := make([]Component, 0, len(r.sorted()))
	for _, e := range r.sorted() {
		out = append(out, e.c)
	}
	return out
}

// Entities is everything the components show Home Assistant.
func (r *Registry) Entities() []esphome.Entity {
	var out []esphome.Entity
	for _, e := range r.sorted() {
		if ents, ok := e.c.(Entities); ok {
			out = append(out, ents.Entities()...)
		}
	}
	return out
}

// Actions is everything the components let Home Assistant call.
func (r *Registry) Actions() []*esphome.Action {
	var out []*esphome.Action
	for _, e := range r.sorted() {
		if acts, ok := e.c.(Actions); ok {
			out = append(out, acts.Actions()...)
		}
	}
	return out
}

// Handlers is the components that answer protocol messages themselves.
func (r *Registry) Handlers() []esphome.Handler {
	var out []esphome.Handler
	for _, e := range r.sorted() {
		if h, ok := e.c.(Handler); ok {
			out = append(out, h)
		}
	}
	return out
}

// Restore puts every component back the way the device was left. Order matters and is the
// registration order: the hardware has to be where it was before anything shows what it is doing.
func (r *Registry) Restore(c config.Config) {
	for _, e := range r.sorted() {
		if v, ok := e.c.(Restorer); ok {
			v.Restore(c)
		}
	}
}

func (r *Registry) Group() *service.Group {
	g := service.New()
	r.AddTo(g)
	return g
}

// AddTo hands the components to a supervisor, in order. It adds rather than owning the group, because
// the group starts in add order and there is still hand-wired work either side.
//
// A component with no loop is added too, as long as it has something to start or close: plenty of
// what a device does at boot happens once and still belongs in the ordered list.
func (r *Registry) AddTo(g *service.Group) {
	for _, e := range r.sorted() {
		if svc, ok := e.c.(service.Service); ok {
			g.Add(svc, e.opts...)
			continue
		}

		_, starts := e.c.(Starter)
		_, closes := e.c.(Closer)
		if !starts && !closes {
			continue
		}
		g.Add(oneshot{e.c}, append(slices.Clone(e.opts), service.Once())...)
	}
}

// oneshot gives a component with no loop the Run the supervisor needs. Returning nil immediately is
// how service.Once reads a service that finished rather than one that died.
//
// Start and Close are forwarded by hand: embedding a Component promotes only the methods of that
// interface, so the wrapper would otherwise hide the very halves it exists to run.
type oneshot struct{ Component }

func (oneshot) Run(context.Context) error { return nil }

func (o oneshot) Start(ctx context.Context) error {
	if s, ok := o.Component.(Starter); ok {
		return s.Start(ctx)
	}
	return nil
}

func (o oneshot) Close() error {
	if c, ok := o.Component.(Closer); ok {
		return c.Close()
	}
	return nil
}

// sorted is the entries in the order everything walks them.
func (r *Registry) sorted() []*entry {
	r.mu.Lock()
	on := r.onBoard()
	out := slices.DeleteFunc(slices.Clone(r.entries), func(e *entry) bool {
		return e.needs&^on.Caps != 0
	})
	r.mu.Unlock()

	// Filtering happens above, before this: the constructor is what opens the hardware, so a component
	// this board does not have must never reach it.
	//
	// Outside the lock: a constructor is the component's own code and has no business being run with
	// the registry held.
	for _, e := range out {
		e.resolve()
	}

	slices.SortStableFunc(out, func(a, b *entry) int {
		if v := cmp.Compare(a.phase, b.phase); v != 0 {
			return v
		}
		if v := cmp.Compare(a.order, b.order); v != 0 {
			return v
		}
		return cmp.Compare(a.c.Name(), b.c.Name())
	})
	return out
}

// Progress is what the components say about coming up, in the order they were brought up. Only
// those that implement Startup have anything to say.
func (r *Registry) Progress() []Progress {
	var out []Progress
	for _, e := range r.sorted() {
		s, ok := e.c.(Startup)
		if !ok {
			continue
		}

		p := s.Startup()
		p.Name = e.c.Name()
		out = append(out, p)
	}
	return out
}

// Ready reports whether everything the boot screen waits for is up.
func (r *Registry) Ready() bool {
	for _, p := range r.Progress() {
		if !p.Settled() {
			return false
		}
	}
	return true
}
