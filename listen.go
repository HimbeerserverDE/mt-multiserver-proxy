package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/anon55555/mt"
)

type listener struct {
	mtListener mt.Listener
	mu         sync.Mutex

	clts map[*clientConn]struct{}
}

func listen(pc net.PacketConn) *listener {
	return &listener{
		mtListener: mt.Listen(pc),
		clts:       make(map[*clientConn]struct{}),
	}
}

func (l *listener) close() error {
	return l.mtListener.Close()
}

func (l *listener) addr() net.Addr { return l.mtListener.Addr() }

func (l *listener) accept() (*clientConn, error) {
	p, err := l.mtListener.Accept()
	if err != nil {
		return nil, err
	}

	cc := &clientConn{
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

	cc.log("-->", "connect")
	go handleClt(cc)

	select {
	case <-cc.Closed():
		return nil, fmt.Errorf("%s is closed", cc.RemoteAddr())
	default:
	}

	return cc, nil
}
