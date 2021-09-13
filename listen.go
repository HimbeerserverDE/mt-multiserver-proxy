package proxy

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/anon55555/mt"
)

var listeners map[*listener]struct{}
var listenersMu sync.RWMutex
var listenersOnce sync.Once

func allListeners() map[*listener]struct{} {
	lm := make(map[*listener]struct{})

	listenersMu.RLock()
	defer listenersMu.RUnlock()

	for l := range listeners {
		lm[l] = struct{}{}
	}

	return lm
}

type listener struct {
	mt.Listener
	mu sync.RWMutex

	clts map[*ClientConn]struct{}
}

func listen(pc net.PacketConn) *listener {
	l := &listener{
		Listener: mt.Listen(pc),
		clts:     make(map[*ClientConn]struct{}),
	}

	listenersMu.Lock()
	defer listenersMu.Unlock()

	listenersOnce.Do(func() {
		listeners = make(map[*listener]struct{})
	})

	listeners[l] = struct{}{}
	return l
}

func (l *listener) clients() map[*ClientConn]struct{} {
	clts := make(map[*ClientConn]struct{})

	l.mu.RLock()
	defer l.mu.RUnlock()

	for cc := range l.clts {
		clts[cc] = struct{}{}
	}

	return clts
}

func (l *listener) accept() (*ClientConn, error) {
	p, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("[%s] ", p.RemoteAddr())
	cc := &ClientConn{
		Peer:   p,
		logger: log.New(logWriter, prefix, log.LstdFlags|log.Lmsgprefix),
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

	cc.Log("->", "connect")
	go handleClt(cc)

	select {
	case <-cc.Closed():
		return nil, fmt.Errorf("%s is closed", cc.RemoteAddr())
	default:
	}

	return cc, nil
}
