package proxy

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

// A ServerConn is a connection to a minetest server.
type ServerConn struct {
	mt.Peer
	clt *ClientConn
	mu  sync.RWMutex

	logger *log.Logger

	cstate   clientState
	cstateMu sync.RWMutex
	name     string
	initCh   chan struct{}

	auth struct {
		method              mt.AuthMethods
		salt, srpA, a, srpK []byte
	}

	mediaPool string

	inv          mt.Inv
	detachedInvs []string

	aos              map[mt.AOID]struct{}
	particleSpawners map[mt.ParticleSpawnerID]struct{}

	sounds map[mt.SoundID]struct{}

	huds map[mt.HUDID]mt.HUDType

	playerList map[string]struct{}
}

func (sc *ServerConn) client() *ClientConn {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.clt
}

func (sc *ServerConn) state() clientState {
	sc.cstateMu.RLock()
	defer sc.cstateMu.RUnlock()

	return sc.cstate
}

func (sc *ServerConn) setState(state clientState) {
	sc.cstateMu.Lock()
	defer sc.cstateMu.Unlock()

	sc.cstate = state
}

// Init returns a channel that is closed
// when the ServerConn enters the csActive state.
func (sc *ServerConn) Init() <-chan struct{} { return sc.initCh }

// Log logs an interaction with the ServerConn.
// dir indicates the direction of the interaction.
func (sc *ServerConn) Log(dir string, v ...interface{}) {
	sc.logger.Println(append([]interface{}{dir}, v...)...)
}

func handleSrv(sc *ServerConn) {
	go func() {
		init := make(chan struct{})
		defer close(init)

		go func(init <-chan struct{}) {
			select {
			case <-init:
			case <-time.After(10 * time.Second):
				sc.Log("->", "timeout")
				sc.Close()
			}
		}(init)

		for sc.state() == csCreated && sc.client() != nil {
			sc.SendCmd(&mt.ToSrvInit{
				SerializeVer: serializeVer,
				MinProtoVer:  protoVer,
				MaxProtoVer:  protoVer,
				PlayerName:   sc.client().Name(),
			})
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for {
		pkt, err := sc.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if errors.Is(sc.WhyClosed(), rudp.ErrTimedOut) {
					sc.Log("<->", "timeout")
				} else {
					sc.Log("<->", "disconnect")
				}

				if sc.client() != nil {
					ack, _ := sc.client().SendCmd(&mt.ToCltKick{
						Reason: mt.Custom,
						Custom: "Server connection closed unexpectedly.",
					})

					select {
					case <-sc.client().Closed():
					case <-ack:
						sc.client().Close()

						sc.client().mu.Lock()
						sc.client().srv = nil
						sc.client().mu.Unlock()

						sc.mu.Lock()
						sc.clt = nil
						sc.mu.Unlock()
					}
				}

				break
			}

			sc.Log("<-", err)
			continue
		}

		sc.process(pkt)
	}
}
