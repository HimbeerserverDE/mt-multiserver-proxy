package main

import (
	"net"

	"github.com/anon55555/mt"
)

func connect(conn net.Conn, name string, cc *clientConn) *serverConn {
	sc := &serverConn{
		Peer:             mt.Connect(conn),
		initCh:           make(chan struct{}),
		clt:              cc,
		name:             name,
		aos:              make(map[mt.AOID]struct{}),
		particleSpawners: make(map[mt.ParticleSpawnerID]struct{}),
		sounds:           make(map[mt.SoundID]struct{}),
		huds:             make(map[mt.HUDID]struct{}),
		playerList:       make(map[string]struct{}),
	}
	sc.log("-->", "connect")
	cc.srv = sc

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
