package proxy

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

type ServerConn struct {
	mt.Peer
	clt *ClientConn
	mu  sync.RWMutex

	cstate   clientState
	cstateMu sync.RWMutex
	name     string
	initCh   chan struct{}

	auth struct {
		method              mt.AuthMethods
		salt, srpA, a, srpK []byte
	}

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

func (sc *ServerConn) Init() <-chan struct{} { return sc.initCh }

func (sc *ServerConn) Log(dir string, v ...interface{}) {
	if sc.client() != nil {
		format := "%s {%s}"
		format += strings.Repeat(" %v", len(v))

		sc.client().Log("", fmt.Sprintf(format, append([]interface{}{
			dir,
			sc.name,
		}, v...)...))
	} else {
		format := "{←|⇶} %s {%s}"
		format += strings.Repeat(" %v", len(v))

		log.Printf(format, append([]interface{}{dir, sc.name}, v...)...)
	}
}

func handleSrv(sc *ServerConn) {
	go func() {
		init := make(chan struct{})
		defer close(init)

		go func(init <-chan struct{}) {
			select {
			case <-init:
			case <-time.After(10 * time.Second):
				sc.Log("-->", "timeout")
				sc.Close()
			}
		}(init)

		for sc.state() == csCreated && sc.client() != nil {
			sc.SendCmd(&mt.ToSrvInit{
				SerializeVer: latestSerializeVer,
				MinProtoVer:  latestProtoVer,
				MaxProtoVer:  latestProtoVer,
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
					ack, _ := sc.client().SendCmd(&mt.ToCltDisco{
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

			sc.Log("<--", err)
			continue
		}

		clt := sc.client()
		if clt == nil {
			sc.Log("<--", "no client")
			continue
		}

		switch cmd := pkt.Cmd.(type) {
		case *mt.ToCltHello:
			if sc.auth.method != 0 {
				sc.Log("<--", "unexpected authentication")
				sc.Close()
				break
			}

			sc.setState(sc.state() + 1)
			if cmd.AuthMethods&mt.FirstSRP != 0 {
				sc.auth.method = mt.FirstSRP
			} else {
				sc.auth.method = mt.SRP
			}

			if cmd.SerializeVer != latestSerializeVer {
				sc.Log("<--", "invalid serializeVer")
				break
			}

			switch sc.auth.method {
			case mt.SRP:
				sc.auth.srpA, sc.auth.a, err = srp.InitiateHandshake()
				if err != nil {
					sc.Log("-->", err)
					break
				}

				sc.SendCmd(&mt.ToSrvSRPBytesA{
					A:      sc.auth.srpA,
					NoSHA1: true,
				})
			case mt.FirstSRP:
				salt, verifier, err := srp.NewClient([]byte(clt.name), []byte{})
				if err != nil {
					sc.Log("-->", err)
					break
				}

				sc.SendCmd(&mt.ToSrvFirstSRP{
					Salt:        salt,
					Verifier:    verifier,
					EmptyPasswd: true,
				})
			default:
				sc.Log("<->", "invalid auth method")
				sc.Close()
			}
		case *mt.ToCltSRPBytesSaltB:
			if sc.auth.method != mt.SRP {
				sc.Log("<--", "multiple authentication attempts")
				break
			}

			sc.auth.srpK, err = srp.CompleteHandshake(sc.auth.srpA, sc.auth.a, []byte(clt.name), []byte{}, cmd.Salt, cmd.B)
			if err != nil {
				sc.Log("-->", err)
				break
			}

			M := srp.ClientProof([]byte(clt.name), cmd.Salt, sc.auth.srpA, cmd.B, sc.auth.srpK)
			if M == nil {
				sc.Log("<--", "SRP safety check fail")
				break
			}

			sc.SendCmd(&mt.ToSrvSRPBytesM{
				M: M,
			})
		case *mt.ToCltDisco:
			sc.Log("<--", "deny access", cmd)
			ack, _ := clt.SendCmd(cmd)

			select {
			case <-clt.Closed():
			case <-ack:
				clt.Close()

				sc.mu.Lock()
				sc.clt = nil
				sc.mu.Unlock()
			}
		case *mt.ToCltAcceptAuth:
			sc.auth = struct {
				method              mt.AuthMethods
				salt, srpA, a, srpK []byte
			}{}
			sc.SendCmd(&mt.ToSrvInit2{Lang: clt.lang})
		case *mt.ToCltDenySudoMode:
			sc.Log("<--", "deny sudo")
		case *mt.ToCltAcceptSudoMode:
			sc.Log("<--", "accept sudo")
			sc.setState(sc.state() + 1)
		case *mt.ToCltAnnounceMedia:
			sc.SendCmd(&mt.ToSrvReqMedia{})

			sc.SendCmd(&mt.ToSrvCltReady{
				Major:    clt.major,
				Minor:    clt.minor,
				Patch:    clt.patch,
				Reserved: clt.reservedVer,
				Version:  clt.versionStr,
				Formspec: clt.formspecVer,
			})

			sc.Log("<->", "handshake completed")
			sc.setState(sc.state() + 1)
			close(sc.initCh)
		case *mt.ToCltInv:
			var oldInv mt.Inv
			copy(oldInv, sc.inv)
			sc.inv.Deserialize(strings.NewReader(cmd.Inv))
			sc.prependInv(sc.inv)

			handStack := mt.Stack{
				Item: mt.Item{
					Name: sc.name + "_hand",
				},
				Count: 1,
			}

			hand := sc.inv.List("hand")
			if hand == nil {
				sc.inv = append(sc.inv, mt.NamedInvList{
					Name: "hand",
					InvList: mt.InvList{
						Width:  0,
						Stacks: []mt.Stack{handStack},
					},
				})
			} else if len(hand.Stacks) == 0 {
				hand.Width = 0
				hand.Stacks = []mt.Stack{handStack}
			}

			b := &strings.Builder{}
			sc.inv.SerializeKeep(b, oldInv)

			clt.SendCmd(&mt.ToCltInv{Inv: b.String()})
		case *mt.ToCltAOMsgs:
			for k := range cmd.Msgs {
				sc.swapAOID(&cmd.Msgs[k].ID)
				sc.handleAOMsg(cmd.Msgs[k].Msg)
			}

			clt.SendCmd(cmd)
		case *mt.ToCltAORmAdd:
			resp := &mt.ToCltAORmAdd{}

			for _, ao := range cmd.Remove {
				delete(sc.aos, ao)
				resp.Remove = append(resp.Remove, ao)
			}

			for _, ao := range cmd.Add {
				if ao.InitData.Name == clt.name {
					clt.currentCAO = ao.ID

					if clt.playerCAO == 0 {
						clt.playerCAO = ao.ID
						for _, msg := range ao.InitData.Msgs {
							sc.handleAOMsg(msg)
						}

						resp.Add = append(resp.Add, ao)
					} else {
						var msgs []mt.IDAOMsg
						for _, msg := range ao.InitData.Msgs {
							msgs = append(msgs, mt.IDAOMsg{
								ID:  ao.ID,
								Msg: msg,
							})
						}

						clt.SendCmd(&mt.ToCltAOMsgs{Msgs: msgs})
					}
				} else {
					sc.swapAOID(&ao.ID)
					for _, msg := range ao.InitData.Msgs {
						sc.handleAOMsg(msg)
					}

					resp.Add = append(resp.Add, ao)
					sc.aos[ao.ID] = struct{}{}
				}
			}

			clt.SendCmd(resp)
		case *mt.ToCltCSMRestrictionFlags:
			cmd.Flags &= ^mt.NoCSMs
			clt.SendCmd(cmd)
		case *mt.ToCltDetachedInv:
			var inv mt.Inv
			inv.Deserialize(strings.NewReader(cmd.Inv))
			sc.prependInv(inv)

			b := &strings.Builder{}
			inv.Serialize(b)

			if cmd.Keep {
				sc.detachedInvs = append(sc.detachedInvs, cmd.Name)
			} else {
				for i, name := range sc.detachedInvs {
					if name == cmd.Name {
						sc.detachedInvs = append(sc.detachedInvs[:i], sc.detachedInvs[i+1:]...)
						break
					}
				}
			}

			clt.SendCmd(&mt.ToCltDetachedInv{
				Name: cmd.Name,
				Keep: cmd.Keep,
				Len:  cmd.Len,
				Inv:  b.String(),
			})
		case *mt.ToCltMediaPush:
			var exit bool
			for _, f := range clt.media {
				if f.name == cmd.Filename {
					exit = true
					break
				}
			}

			if exit {
				break
			}

			prepend(sc.name, &cmd.Filename)
			clt.SendCmd(cmd)
		case *mt.ToCltSkyParams:
			for i := range cmd.Textures {
				prependTexture(sc.name, &cmd.Textures[i])
			}
			clt.SendCmd(cmd)
		case *mt.ToCltSunParams:
			prependTexture(sc.name, &cmd.Texture)
			prependTexture(sc.name, &cmd.ToneMap)
			prependTexture(sc.name, &cmd.Rise)
			clt.SendCmd(cmd)
		case *mt.ToCltMoonParams:
			prependTexture(sc.name, &cmd.Texture)
			prependTexture(sc.name, &cmd.ToneMap)
			clt.SendCmd(cmd)
		case *mt.ToCltSetHotbarParam:
			prependTexture(sc.name, &cmd.Img)
			clt.SendCmd(cmd)
		case *mt.ToCltUpdatePlayerList:
			if !clt.playerListInit {
				clt.playerListInit = true
			} else if cmd.Type == mt.InitPlayers {
				cmd.Type = mt.AddPlayers
			}

			if cmd.Type <= mt.AddPlayers {
				for _, player := range cmd.Players {
					sc.playerList[player] = struct{}{}
				}
			} else if cmd.Type == mt.RemovePlayers {
				for _, player := range cmd.Players {
					delete(sc.playerList, player)
				}
			}

			clt.SendCmd(cmd)
		case *mt.ToCltSpawnParticle:
			prependTexture(sc.name, &cmd.Texture)
			sc.globalParam0(&cmd.NodeParam0)
			clt.SendCmd(cmd)
		case *mt.ToCltBlkData:
			for i := range cmd.Blk.Param0 {
				sc.globalParam0(&cmd.Blk.Param0[i])
			}

			for k := range cmd.Blk.NodeMetas {
				for j, field := range cmd.Blk.NodeMetas[k].Fields {
					if field.Name == "formspec" {
						sc.prependFormspec(&cmd.Blk.NodeMetas[k].Fields[j].Value)
						break
					}
				}
				sc.prependInv(cmd.Blk.NodeMetas[k].Inv)
			}

			clt.SendCmd(cmd)
		case *mt.ToCltAddNode:
			sc.globalParam0(&cmd.Node.Param0)
			clt.SendCmd(cmd)
		case *mt.ToCltAddParticleSpawner:
			prependTexture(sc.name, &cmd.Texture)
			sc.swapAOID(&cmd.AttachedAOID)
			sc.globalParam0(&cmd.NodeParam0)
			sc.particleSpawners[cmd.ID] = struct{}{}

			clt.SendCmd(cmd)
		case *mt.ToCltDelParticleSpawner:
			delete(sc.particleSpawners, cmd.ID)
			clt.SendCmd(cmd)
		case *mt.ToCltPlaySound:
			prepend(sc.name, &cmd.Name)
			sc.swapAOID(&cmd.SrcAOID)
			if cmd.Loop {
				sc.sounds[cmd.ID] = struct{}{}
			}

			clt.SendCmd(cmd)
		case *mt.ToCltFadeSound:
			delete(sc.sounds, cmd.ID)
			clt.SendCmd(cmd)
		case *mt.ToCltStopSound:
			delete(sc.sounds, cmd.ID)
			clt.SendCmd(cmd)
		case *mt.ToCltAddHUD:
			sc.prependHUD(cmd.Type, cmd)

			sc.huds[cmd.ID] = cmd.Type
			clt.SendCmd(cmd)
		case *mt.ToCltChangeHUD:
			sc.prependHUD(sc.huds[cmd.ID], cmd)
			clt.SendCmd(cmd)
		case *mt.ToCltRmHUD:
			delete(sc.huds, cmd.ID)
			clt.SendCmd(cmd)
		case *mt.ToCltShowFormspec:
			sc.prependFormspec(&cmd.Formspec)
			clt.SendCmd(cmd)
		case *mt.ToCltFormspecPrepend:
			sc.prependFormspec(&cmd.Prepend)
			clt.SendCmd(cmd)
		case *mt.ToCltInvFormspec:
			sc.prependFormspec(&cmd.Formspec)
			clt.SendCmd(cmd)
		case *mt.ToCltMinimapModes:
			for i := range cmd.Modes {
				prependTexture(sc.name, &cmd.Modes[i].Texture)
			}
			clt.SendCmd(cmd)
		case *mt.ToCltNodeMetasChanged:
			for k := range cmd.Changed {
				for i, field := range cmd.Changed[k].Fields {
					if field.Name == "formspec" {
						sc.prependFormspec(&cmd.Changed[k].Fields[i].Value)
						break
					}
				}
				sc.prependInv(cmd.Changed[k].Inv)
			}
			clt.SendCmd(cmd)
		case *mt.ToCltAddPlayerVel:
			clt.SendCmd(cmd)
		case *mt.ToCltBreath:
			clt.SendCmd(cmd)
		case *mt.ToCltChatMsg:
			clt.SendCmd(cmd)
		case *mt.ToCltCloudParams:
			clt.SendCmd(cmd)
		case *mt.ToCltDeathScreen:
			clt.SendCmd(cmd)
		case *mt.ToCltEyeOffset:
			clt.SendCmd(cmd)
		case *mt.ToCltFOV:
			clt.SendCmd(cmd)
		case *mt.ToCltHP:
			clt.SendCmd(cmd)
		case *mt.ToCltHUDFlags:
			clt.SendCmd(cmd)
		case *mt.ToCltLocalPlayerAnim:
			clt.SendCmd(cmd)
		case *mt.ToCltModChanMsg:
			clt.SendCmd(cmd)
		case *mt.ToCltModChanSig:
			var exit bool
			switch cmd.Signal {
			case mt.JoinOK:
				if _, ok := clt.modChs[cmd.Channel]; ok {
					exit = true
					break
				}
				clt.modChs[cmd.Channel] = struct{}{}
			case mt.JoinFail:
				fallthrough
			case mt.LeaveOK:
				delete(clt.modChs, cmd.Channel)
			}

			if exit {
				break
			}

			clt.SendCmd(cmd)
		case *mt.ToCltMovePlayer:
			clt.SendCmd(cmd)
		case *mt.ToCltMovement:
			clt.SendCmd(cmd)
		case *mt.ToCltOverrideDayNightRatio:
			clt.SendCmd(cmd)
		case *mt.ToCltPrivs:
			clt.SendCmd(cmd)
		case *mt.ToCltRemoveNode:
			clt.SendCmd(cmd)
		case *mt.ToCltStarParams:
			clt.SendCmd(cmd)
		case *mt.ToCltTimeOfDay:
			clt.SendCmd(cmd)
		}
	}
}
