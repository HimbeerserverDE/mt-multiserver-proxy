package main

import (
	"net"

	"github.com/anon55555/mt"
)

func connect(conn net.Conn) *serverConn {
	sc := &serverConn{
		Peer: mt.Connect(conn),
	}

	go handleSrv(sc)
	return sc
}

func connectContent(conn net.Conn, name, userName string) *contentConn {
	cc := &contentConn{
		Peer:     mt.Connect(conn),
		doneCh:   make(chan struct{}),
		name:     name,
		userName: userName,
	}

	go handleContent(cc)
	return cc
}
