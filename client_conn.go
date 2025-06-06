package proxy

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/HimbeerserverDE/mt"
)

type clientState uint8

const (
	csCreated clientState = iota
	csInit
	csActive
	csSudo
)

// A ClientConn is a connection to a minetest client.
type ClientConn struct {
	mt.Peer
	created time.Time
	srv     *ServerConn
	mu      sync.RWMutex

	logger *log.Logger

	cstate   clientState
	cstateMu sync.RWMutex
	name     string
	initCh   chan struct{}
	hopMu    sync.Mutex

	auth struct {
		method                       mt.AuthMethods
		salt, srpA, srpB, srpM, srpK []byte
	}
	new bool

	fallbackFrom string
	whyKicked    *mt.ToCltKick

	lang string

	major, minor, patch uint8
	reservedVer         uint8
	versionStr          string
	formspecVer         uint16

	denyPools map[string]struct{}
	itemDefs  []mt.ItemDef
	aliases   []struct{ Alias, Orig string }
	nodeDefs  []mt.NodeDef
	p0Map     param0Map
	p0SrvMap  param0SrvMap
	media     []mediaFile

	playerCAO, currentCAO mt.AOID

	playerListInit bool

	modChs   map[string]struct{}
	modChsMu sync.RWMutex

	cltInfo *mt.ToSrvCltInfo

	FormspecPrepend string
}

// Name returns the player name of the ClientConn.
func (cc *ClientConn) Name() string { return cc.name }

// IsNew reports whether a new account was registered for the ClientConn.
func (cc *ClientConn) IsNew() bool { return cc.new }

func (cc *ClientConn) hasPlayerCAO() bool { return cc.playerCAO != 0 }

func (cc *ClientConn) server() *ServerConn {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.srv
}

// ServerName returns the name of the current upstream server
// of the ClientConn. It is empty if there is no upstream connection.
func (cc *ClientConn) ServerName() string {
	srv := cc.server()
	if srv != nil {
		return srv.name
	}

	return ""
}

func (cc *ClientConn) state() clientState {
	cc.cstateMu.RLock()
	defer cc.cstateMu.RUnlock()

	return cc.cstate
}

func (cc *ClientConn) setState(state clientState) {
	cc.cstateMu.Lock()
	defer cc.cstateMu.Unlock()

	cc.cstate = state
}

// Init returns a channel that is closed
// when the ClientConn enters the csActive state.
func (cc *ClientConn) Init() <-chan struct{} { return cc.initCh }

// Log logs an interaction with the ClientConn.
// dir indicates the direction of the interaction.
func (cc *ClientConn) Log(dir string, v ...interface{}) {
	cc.logger.Println(append([]interface{}{dir}, v...)...)
}

// CltInfo returns the ToSrvCltInfo known about the client.
func (cc *ClientConn) ToSrvCltInfo() *mt.ToSrvCltInfo { return cc.cltInfo }

func handleClt(cc *ClientConn) {
	for {
		pkt, err := cc.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if cc.state() == csActive {
					handleLeave(cc)
				}

				if why := cc.WhyClosed(); why != nil {
					cc.Log("<->", "connection lost:", why)
				} else {
					cc.Log("<->", "disconnect")
				}

				if cc.Name() != "" {
					playersMu.Lock()
					delete(players, cc.Name())
					playersMu.Unlock()
				}

				cc.mu.Lock()
				if cc.srv != nil {
					cc.srv.mu.Lock()
					cc.srv.clt = nil
					cc.srv.mu.Unlock()

					cc.srv.Close()
					cc.srv = nil
				}
				cc.mu.Unlock()

				break
			}

			cc.Log("->", err)
			continue
		}

		cc.process(pkt)
	}
}
