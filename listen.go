package proxy

import (
	"fmt"
	"net"
	"sync"

	"github.com/anon55555/mt"
)

var listeners map[*Listener]struct{}
var listenersMu sync.RWMutex
var listenersOnce sync.Once

func Listeners() map[*Listener]struct{} {
	lm := make(map[*Listener]struct{})

	listenersMu.RLock()
	defer listenersMu.RUnlock()

	for l := range listeners {
		lm[l] = struct{}{}
	}

	return lm
}

type Listener struct {
	mt.Listener
	mu sync.RWMutex

	clts map[*ClientConn]struct{}
}

func Listen(pc net.PacketConn) *Listener {
	l := &Listener{
		Listener: mt.Listen(pc),
		clts:     make(map[*ClientConn]struct{}),
	}

	listenersMu.Lock()
	defer listenersMu.Unlock()

	listenersOnce.Do(func() {
		listeners = make(map[*Listener]struct{})
	})

	listeners[l] = struct{}{}
	return l
}

func (l *Listener) Clts() map[*ClientConn]struct{} {
	clts := make(map[*ClientConn]struct{})

	l.mu.RLock()
	defer l.mu.RUnlock()

	for cc := range l.clts {
		clts[cc] = struct{}{}
	}

	return clts
}

func (l *Listener) Accept() (*ClientConn, error) {
	p, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	cc := &ClientConn{
		Peer:   p,
		initCh: make(chan struct{}),
		modChs: make(map[string]struct{}),
	}

	l.mu.Lock()
	l.clts[cc] = struct{}{}
	l.mu.Unlock()

	go func() {
		<-cc.Closed()
		l.mu.Lock()
		defer l.mu.Unlock()

		delete(l.clts, cc)
	}()

	cc.Log("-->", "connect")
	go handleClt(cc)

	select {
	case <-cc.Closed():
		return nil, fmt.Errorf("%s is closed", cc.RemoteAddr())
	default:
	}

	return cc, nil
}
