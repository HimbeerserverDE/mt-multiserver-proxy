package main

import (
	"fmt"
	"net"

	"github.com/anon55555/mt"
)

type listener struct {
	mtListener mt.Listener
}

func listen(pc net.PacketConn) *listener {
	return &listener{
		mtListener: mt.Listen(pc),
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
	}

	cc.log("-->", "connect")
	go handleClt(cc)

	select {
	case <-cc.Closed():
		return nil, fmt.Errorf("%s is closed", cc.RemoteAddr())
	default:
	}

	return cc, nil
}
