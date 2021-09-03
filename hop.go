package main

import (
	"fmt"
	"net"

	"github.com/anon55555/mt"
)

func (cc *clientConn) hop(serverName string) error {
	cc.hopMu.Lock()
	defer cc.hopMu.Unlock()

	if cc.server() == nil {
		err := fmt.Errorf("hop: no server connection")
		cc.log("<->", err)
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

	// Static parameters
	cc.SendCmd(&mt.ToCltBreath{Breath: 10})
	cc.SendCmd(&mt.ToCltCloudParams{})
	cc.SendCmd(&mt.ToCltEyeOffset{})
	cc.SendCmd(&mt.ToCltFOV{})
	cc.SendCmd(&mt.ToCltFormspecPrepend{})
	cc.SendCmd(&mt.ToCltHP{})
	cc.SendCmd(&mt.ToCltHUDFlags{Mask: ^mt.HUDFlags(0)})
	cc.SendCmd(&mt.ToCltLocalPlayerAnim{})
	cc.SendCmd(&mt.ToCltMinimapModes{})
	cc.SendCmd(&mt.ToCltMoonParams{})
	cc.SendCmd(&mt.ToCltMovement{})
	cc.SendCmd(&mt.ToCltOverrideDayNightRatio{})
	cc.SendCmd(&mt.ToCltPrivs{})

	for i := mt.HotbarParam(mt.HotbarSize); i < mt.HotbarSelImg; i++ {
		cc.SendCmd(&mt.ToCltSetHotbarParam{Param: i})
	}

	cc.SendCmd(&mt.ToCltSkyParams{})
	cc.SendCmd(&mt.ToCltStarParams{})
	cc.SendCmd(&mt.ToCltSunParams{})

	var players []string
	for player := range cc.server().playerList {
		players = append(players, player)
	}

	cc.SendCmd(&mt.ToCltUpdatePlayerList{
		Type:    mt.RemovePlayers,
		Players: players,
	})

	cc.srv = nil

	addr, err := net.ResolveUDPAddr("udp", strAddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	connect(conn, serverName, cc)

	for ch := range cc.modChs {
		cc.server().SendCmd(&mt.ToSrvJoinModChan{Channel: ch})
	}

	return nil
}
