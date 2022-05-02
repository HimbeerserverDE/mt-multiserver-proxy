package proxy

import (
	"fmt"
	"log"
	"net"

	"github.com/anon55555/mt"
)

func connect(conn net.Conn, name string, cc *ClientConn) *ServerConn {
	cc.mu.RLock()
	if cc.srv != nil {
		cc.Log("<->", "already connected to server")
		cc.mu.RUnlock()
		return nil
	}
	cc.mu.RUnlock()

	var mediaPool string
	for srvName, srv := range Conf().Servers {
		if srvName == name {
			mediaPool = srv.MediaPool
		}
	}

	logPrefix := fmt.Sprintf("[server %s] ", name)
	sc := &ServerConn{
		Peer:             mt.Connect(conn),
		logger:           log.New(logWriter, logPrefix, log.LstdFlags|log.Lmsgprefix),
		initCh:           make(chan struct{}),
		clt:              cc,
		name:             name,
		mediaPool:        mediaPool,
		aos:              make(map[mt.AOID]struct{}),
		particleSpawners: make(map[mt.ParticleSpawnerID]struct{}),
		sounds:           make(map[mt.SoundID]struct{}),
		huds:             make(map[mt.HUDID]mt.HUDType),
		playerList:       make(map[string]struct{}),
	}
	sc.Log("->", "connect")

	cc.mu.Lock()
	cc.srv = sc
	cc.mu.Unlock()

	go handleSrv(sc)
	return sc
}

func connectContent(conn net.Conn, name, userName, mediaPool string) (*contentConn, error) {
	logPrefix := fmt.Sprintf("[content %s] ", name)
	cc := &contentConn{
		Peer:      mt.Connect(conn),
		logger:    log.New(logWriter, logPrefix, log.LstdFlags|log.Lmsgprefix),
		doneCh:    make(chan struct{}),
		name:      name,
		userName:  userName,
		mediaPool: mediaPool,
	}

	if err := cc.addDefaultTextures(); err != nil {
		return nil, err
	}

	go handleContent(cc)
	return cc, nil
}
