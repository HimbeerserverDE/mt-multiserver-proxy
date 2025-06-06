package proxy

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/HimbeerserverDE/mt"
	"github.com/HimbeerserverDE/mt/rudp"
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
	dynMedia  map[string]struct {
		ephemeral bool
		token     uint32
	}

	inv          mt.Inv
	detachedInvs []string

	aos              map[mt.AOID]struct{}
	particleSpawners map[mt.ParticleSpawnerID]struct{}

	sounds map[mt.SoundID]struct{}

	huds map[mt.HUDID]mt.HUDType

	playerList map[string]struct{}

	modChanJoinChs   map[string]map[chan bool]struct{}
	modChanJoinChMu  sync.Mutex
	modChanLeaveChs  map[string]map[chan bool]struct{}
	modChanLeaveChMu sync.Mutex
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
				if why := sc.WhyClosed(); why != nil {
					sc.Log("<->", "connection lost:", why)
				} else {
					sc.Log("<->", "disconnect")
				}

				sc.mu.Lock()
				if sc.clt != nil {
					if errors.Is(sc.WhyClosed(), rudp.ErrTimedOut) {
						sc.clt.SendChatMsg("Server connection timed out, switching to fallback server.")

						if sc.clt.whyKicked == nil {
							sc.clt.whyKicked = &mt.ToCltKick{
								Reason: mt.Custom,
								Custom: "Server connection timed out.",
							}
						}
					} else {
						sc.clt.SendChatMsg("Server connection lost, switching to fallback server.")

						if sc.clt.whyKicked == nil {
							sc.clt.whyKicked = &mt.ToCltKick{
								Reason: mt.Custom,
								Custom: "Server connection lost.",
							}
						}
					}

					sc.clt.fallback()
				}
				sc.mu.Unlock()

				break
			}

			sc.Log("<-", err)
			continue
		}

		sc.process(pkt)
	}
}
