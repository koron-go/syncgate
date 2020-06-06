package syncgate

import (
	"context"
	"fmt"
	"sync"
)

// Gate is mutiple waiting mutex objects.
type Gate struct {
	name string

	mu     sync.Mutex
	opened bool
	chans  map[chan<- *Gate]struct{}
}

// New creates a Gate object. Only first string in `name` is used as Gate's
// name. If `name` is omitted, Gate will have no name.
func New(name ...string) *Gate {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	return &Gate{name: n}
}

// Open opens a gate.
func (g *Gate) Open() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.opened {
		return
	}
	g.opened = true
	for ch := range g.chans {
		ch <- g
	}
	g.chans = nil
}

// IsOpen checks a Gate has been opened or not.
func (g *Gate) IsOpen() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.opened
}

func (g *Gate) String() string {
	return fmt.Sprintf("syncgate.Gate{ name=%s opened=%t }", g.name, g.opened)
}

func (g *Gate) register(ch chan<- *Gate) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.opened {
		ch <- g
		return
	}
	if g.chans == nil {
		g.chans = make(map[chan<- *Gate]struct{})
	}
	g.chans[ch] = struct{}{}
}

func (g *Gate) unregister(ch chan<- *Gate) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.chans == nil {
		return
	}
	delete(g.chans, ch)
}

// Wait waits all gates are opened or context.Context canceled.
func Wait(ctx context.Context, gates ...*Gate) error {
	n := len(gates)
	if n == 0 {
		return nil
	}
	ch := make(chan *Gate, len(gates))
	for _, g := range gates {
		g.register(ch)
	}
	for {
		select {
		case <-ch:
			n--
			if n <= 0 {
				close(ch)
				return nil
			}
		case <-ctx.Done():
			close(ch)
			for _, g := range gates {
				g.unregister(ch)
			}
			return ctx.Err()
		}
	}
}

// Try checks all gates are opened or not.
func Try(gates ...*Gate) bool {
	if len(gates) == 0 {
		return true
	}
	for _, g := range gates {
		if !g.opened {
			return false
		}
	}
	return true
}
