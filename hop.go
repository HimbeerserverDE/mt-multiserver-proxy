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
// or the first working fallback server, saving the player's last server
// unless `ForceDefaultSrv` is enabled.
// If all attempts fail the client stays connected to the current server
// with the potential for inconsistent state.
// At the moment the ClientConn is NOT fixed if an error occurs
// so the player may have to reconnect.
func (cc *ClientConn) Hop(serverName string) (err error) {
	defer func() {
		if err == nil && !Conf().ForceDefaultSrv {
			err = DefaultAuth().SetLastSrv(cc.Name(), serverName)
		}
	}()

	err = cc.HopRaw(serverName)
	return
}

// HopGroup connects the ClientConn to the specified server group
// or the first working fallback server, saving the player's last server
// unless `ForceDefaultSrv` is enabled.
// See the documentation on `Server.Groups` in `doc/config.md`
// for details on how a specific game server is selected from the group name.
// If all attempts fail the client stays connected to the current server
// with the potential for inconsistent state.
// At the moment the ClientConn is NOT fixed if an error occurs
// so the player may have to reconnect.
func (cc *ClientConn) HopGroup(groupName string) error {
	choice, ok := Conf().RandomGroupServer(groupName)
	if !ok {
		return ErrNoSuchServer
	}

	return cc.Hop(choice)
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

	if _, ok := cc.denyPools[newSrv.MediaPool]; ok {
		return ErrNewMediaPool
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
	if cc.hasPlayerCAO() {
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
		cc.SendCmd(&mt.ToCltCloudParams{
			Density:      0.4,
			DiffuseColor: color.NRGBA{229, 240, 240, 255},
			AmbientColor: color.NRGBA{255, 0, 0, 0},
			ShadowColor:  color.NRGBA{255, 204, 204, 204},
			Height:       120,
			Thickness:    16,
			Speed:        [2]float32{0, -2},
		})
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
		cc.SendCmd(&mt.ToCltCam{Mode: mt.AnyCam})
		cc.SendCmd(&mt.ToCltOverrideDayNightRatio{})
		cc.SendCmd(&mt.ToCltPrivs{})

		for i := mt.HotbarParam(mt.HotbarSize); i <= mt.HotbarSelImg; i++ {
			cc.SendCmd(&mt.ToCltSetHotbarParam{Param: i})
		}

		cc.SendCmd(&mt.ToCltSkyParams{
			BgColor:       color.NRGBA{255, 255, 255, 255},
			Type:          "regular",
			Clouds:        true,
			SunFogTint:    color.NRGBA{244, 125, 29, 255},
			MoonFogTint:   color.NRGBA{128, 153, 204, 255},
			FogTintType:   "default",
			DayHorizon:    color.NRGBA{144, 211, 246, 255},
			DawnHorizon:   color.NRGBA{186, 193, 240, 255},
			NightHorizon:  color.NRGBA{64, 144, 255, 255},
			DaySky:        color.NRGBA{97, 181, 245, 255},
			DawnSky:       color.NRGBA{180, 186, 250, 255},
			NightSky:      color.NRGBA{0, 107, 255, 255},
			Indoor:        color.NRGBA{100, 100, 100, 255},
			BodyOrbitTilt: -1024,
			FogDistance:   -1,
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

		cc.SendCmd(&mt.ToCltLighting{
			Saturation:     1,
			ShadowTint:     color.NRGBA{255, 0, 0, 0},
			BloomIntensity: 0.05,
			BloomStrength:  1,
			BloomRadius:    1,
		})
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

	if cc.cltInfo != nil { // May not be initialized yet if this is an early fallback.
		cc.server().SendCmd(cc.cltInfo)
	}

	return nil
}
