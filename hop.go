package proxy

import (
	"errors"
	"image/color"
	"net"

	"github.com/HimbeerserverDE/mt"
)

var (
	ErrNoServerConn = errors.New("no server connection")
	ErrNoSuchServer = errors.New("inexistent server")
	ErrNewMediaPool = errors.New("media pool unknown to client")
)

// Hop connects the ClientConn to the specified upstream server
// or the first working fallback server, saving the player's last server.
// If all attempts fail the client is kicked.
// At the moment the ClientConn is NOT fixed if an error occurs
// so the player may have to reconnect.
func (cc *ClientConn) Hop(serverName string) (err error) {
	defer func() {
		if !Conf().ForceDefaultSrv {
			err = authIface.SetLastSrv(cc.Name(), serverName)
		}
	}()

	if err = cc.HopRaw(serverName); err != nil {
		if errors.Is(err, ErrNoSuchServer) || errors.Is(err, ErrNewMediaPool) {
			return err
		}

		cc.Log("<-", err)
		cc.SendChatMsg("Could not switch servers, triggering fallback. Error:", err.Error())

		for _, srvName := range FallbackServers(serverName) {
			if err = cc.HopRaw(srvName); err != nil {
				cc.Log("<-", err)
				cc.SendChatMsg("Could not connect, continuing fallback. Error:", err.Error())
			}

			return nil
		}

		return err
	}

	return nil
}

// HopRaw connects the ClientConn to the specified upstream server.
// At the moment the ClientConn is NOT fixed if an error occurs
// so the player may have to reconnect.
//
// This method ignores fallback servers and doesn't save the player's
// last server.
// You may use the `Hop` wrapper for these purposes.
func (cc *ClientConn) HopRaw(serverName string) error {
	cc.hopMu.Lock()
	defer cc.hopMu.Unlock()

	cc.Log("<->", "hop", serverName)

	if cc.server() == nil {
		return ErrNoServerConn
	}

	newSrv, ok := Conf().Servers[serverName]
	if !ok {
		return ErrNoSuchServer
	}

	if newSrv.poolAdded.After(cc.created) {
		return ErrNewMediaPool
	}

	// This needs to be done before the ServerConn is closed
	// so the clientConn isn't closed by the packet handler
	cc.server().mu.Lock()
	cc.server().clt = nil
	cc.server().mu.Unlock()

	cc.server().Close()

	// Player CAO is a good indicator for full client initialization.
	if !cc.hasPlayerCAO() {
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
		cc.SendCmd(&mt.ToCltHUDFlags{Mask: ^mt.HUDFlags(0), Flags: ^mt.HUDFlags(0)})
		cc.SendCmd(&mt.ToCltLocalPlayerAnim{})

		cc.SendCmd(&mt.ToCltMinimapModes{
			Modes: mt.DefaultMinimap,
		})

		cc.SendCmd(&mt.ToCltMoonParams{
			Visible: true,
			Texture: "moon.png",
			ToneMap: "moon_tonemap.png",
			Size:    1,
		})

		cc.SendCmd(&mt.ToCltMovement{})
		cc.SendCmd(&mt.ToCltOverrideDayNightRatio{})
		cc.SendCmd(&mt.ToCltPrivs{})

		for i := mt.HotbarParam(mt.HotbarSize); i <= mt.HotbarSelImg; i++ {
			cc.SendCmd(&mt.ToCltSetHotbarParam{Param: i})
		}

		cc.SendCmd(&mt.ToCltSkyParams{
			Type:         "regular",
			Clouds:       true,
			DayHorizon:   color.NRGBA{144, 211, 246, 255},
			DawnHorizon:  color.NRGBA{186, 193, 240, 255},
			NightHorizon: color.NRGBA{64, 144, 255, 255},
			DaySky:       color.NRGBA{97, 181, 245, 255},
			DawnSky:      color.NRGBA{180, 186, 250, 255},
			NightSky:     color.NRGBA{0, 107, 255, 255},
			Indoor:       color.NRGBA{100, 100, 100, 255},
		})

		cc.SendCmd(&mt.ToCltStarParams{
			Visible: true,
			Count:   1000,
			Color:   color.NRGBA{105, 235, 235, 255},
			Size:    1,
		})

		cc.SendCmd(&mt.ToCltSunParams{
			Visible: true,
			Texture: "sun.png",
			ToneMap: "sun_tonemap.png",
			Rise:    "sunrisebg.png",
			Rising:  true,
			Size:    1,
		})

		var players []string
		for player := range cc.server().playerList {
			players = append(players, player)
		}

		cc.SendCmd(&mt.ToCltUpdatePlayerList{
			Type:    mt.RemovePlayers,
			Players: players,
		})

		cc.SendCmd(&mt.ToCltLighting{Saturation: 1})
	}

	cc.mu.Lock()
	cc.srv = nil
	cc.mu.Unlock()

	addr, err := net.ResolveUDPAddr("udp", newSrv.Addr)
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

	cc.server().SendCmd(cc.cltInfo)

	return nil
}
