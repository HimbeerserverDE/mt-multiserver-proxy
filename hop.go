package main

import (
	"fmt"

	"github.com/anon55555/mt"
)

func (cc *clientConn) hop(serverName string) error {
	cc.hopMu.Lock()
	defer cc.hopMu.Unlock()

	if cc.server() == nil {
		err := fmt.Errorf("hop: no server connection")
		cc.log("<->", err.Error())
		return err
	}

	var strAddr string
	for _, srv := range conf.Servers {
		if srv.Name == serverName {
			strAddr = srv.Addr
			break
		}
	}

	if strAddr == "" {
		return fmt.Errorf("hop: inexistent server")
	}

	// This needs to be done before the serverConn is closed
	// so the clientConn isn't closed by the packet handler
	cc.server().clt = nil
	cc.server().Close()

	// Reset the client to its initial state
	for _, inv := range cc.server().detachedInvs {
		cc.SendCmd(&mt.ToCltDetachedInv{
			Name: inv,
			Keep: false,
		})
	}

	var aoRm []mt.AOID
	for ao := range cc.server().aos {
		aoRm = append(aoRm, ao)
	}
	cc.SendCmd(&mt.ToCltAORmAdd{Remove: aoRm})

	for spawner := range cc.server().particleSpawners {
		cc.SendCmd(&mt.ToCltDelParticleSpawner{ID: spawner})
	}

	for sound := range cc.server().sounds {
		cc.SendCmd(&mt.ToCltStopSound{ID: sound})
	}

	for hud := range cc.server().huds {
		cc.SendCmd(&mt.ToCltRmHUD{ID: hud})
	}

	// Stateless packets
	return nil
}
