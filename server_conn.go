package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

type serverConn struct {
	mt.Peer
	clt *clientConn

	state  clientState
	name   string
	initCh chan struct{}

	auth struct {
		method              mt.AuthMethods
		salt, srpA, a, srpK []byte
	}

	inv          mt.Inv
	detachedInvs []string

	aos map[mt.AOID]struct{}
}

func (sc *serverConn) client() *clientConn { return sc.clt }

func (sc *serverConn) init() <-chan struct{} { return sc.initCh }

func (sc *serverConn) log(dir, msg string) {
	if sc.client() != nil {
		sc.client().log("", fmt.Sprintf("%s {%s} %s", dir, sc.name, msg))
	} else {
		log.Printf("{←|⇶} %s {%s} %s", dir, sc.name, msg)
	}
}

func handleSrv(sc *serverConn) {
	if sc.client() == nil {
		sc.log("-->", "no associated client")
	}

	go func() {
		for sc.state == csCreated && sc.client() != nil {
			sc.SendCmd(&mt.ToSrvInit{
				SerializeVer: latestSerializeVer,
				MinProtoVer:  latestProtoVer,
				MaxProtoVer:  latestProtoVer,
				PlayerName:   sc.client().name,
			})
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for {
		pkt, err := sc.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if errors.Is(sc.WhyClosed(), rudp.ErrTimedOut) {
					sc.log("<->", "timeout")
				} else {
					sc.log("<->", "disconnect")
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
						sc.client().srv = nil
						sc.clt = nil
					}
				}

				break
			}

			sc.log("-->", err.Error())
			continue
		}

		switch cmd := pkt.Cmd.(type) {
		case *mt.ToCltHello:
			if sc.auth.method != 0 {
				sc.log("<--", "unexpected authentication")
				sc.Close()
				break
			}

			sc.state++

			if cmd.AuthMethods&mt.FirstSRP != 0 {
				sc.auth.method = mt.FirstSRP
			} else {
				sc.auth.method = mt.SRP
			}

			if cmd.SerializeVer != latestSerializeVer {
				sc.log("<--", "invalid serializeVer")
				break
			}

			switch sc.auth.method {
			case mt.SRP:
				sc.auth.srpA, sc.auth.a, err = srp.InitiateHandshake()
				if err != nil {
					sc.log("-->", err.Error())
					break
				}

				sc.SendCmd(&mt.ToSrvSRPBytesA{
					A:      sc.auth.srpA,
					NoSHA1: true,
				})
			case mt.FirstSRP:
				salt, verifier, err := srp.NewClient([]byte(sc.client().name), []byte{})
				if err != nil {
					sc.log("-->", err.Error())
					break
				}

				sc.SendCmd(&mt.ToSrvFirstSRP{
					Salt:        salt,
					Verifier:    verifier,
					EmptyPasswd: true,
				})
			default:
				sc.log("<->", "invalid auth method")
				sc.Close()
			}
		case *mt.ToCltSRPBytesSaltB:
			if sc.auth.method != mt.SRP {
				sc.log("<--", "multiple authentication attempts")
				break
			}

			sc.auth.srpK, err = srp.CompleteHandshake(sc.auth.srpA, sc.auth.a, []byte(sc.client().name), []byte{}, cmd.Salt, cmd.B)
			if err != nil {
				sc.log("-->", err.Error())
				break
			}

			M := srp.ClientProof([]byte(sc.client().name), cmd.Salt, sc.auth.srpA, cmd.B, sc.auth.srpK)
			if M == nil {
				sc.log("<--", "SRP safety check fail")
				break
			}

			sc.SendCmd(&mt.ToSrvSRPBytesM{
				M: M,
			})
		case *mt.ToCltDisco:
			sc.log("<--", fmt.Sprintf("deny access %+v", cmd))
			ack, _ := sc.client().SendCmd(cmd)

			select {
			case <-sc.client().Closed():
			case <-ack:
				sc.client().Close()
				sc.clt = nil
			}
		case *mt.ToCltAcceptAuth:
			sc.auth.method = 0
			sc.SendCmd(&mt.ToSrvInit2{Lang: sc.client().lang})
		case *mt.ToCltDenySudoMode:
			sc.log("<--", "deny sudo")
		case *mt.ToCltAcceptSudoMode:
			sc.log("<--", "accept sudo")
			sc.state++
		case *mt.ToCltAnnounceMedia:
			sc.SendCmd(&mt.ToSrvReqMedia{})

			sc.SendCmd(&mt.ToSrvCltReady{
				Major:    sc.client().major,
				Minor:    sc.client().minor,
				Patch:    sc.client().patch,
				Reserved: sc.client().reservedVer,
				Version:  sc.client().versionStr,
				Formspec: sc.client().formspecVer,
			})

			sc.log("<->", "handshake completed")
			sc.state++
			close(sc.initCh)
		case *mt.ToCltInv:
			var inv mt.Inv
			inv.Deserialize(strings.NewReader(cmd.Inv))

			for k, l := range inv {
				for i := range l.Stacks {
					prepend(sc.name, &inv[k].InvList.Stacks[i].Name)
				}
			}

			var t mt.ToolCaps
			for _, iDef := range sc.client().itemDefs {
				if iDef.Name == sc.name+"_hand" {
					t = iDef.ToolCaps
					break
				}
			}

			var tc ToolCaps
			tc.fromMT(t)

			b := &strings.Builder{}
			tc.SerializeJSON(b)

			fields := []mt.Field{
				{
					Name:  "tool_capabilities",
					Value: b.String(),
				},
			}
			meta := mt.NewItemMeta(fields)

			handStack := mt.Stack{
				Item: mt.Item{
					Name:     sc.name + "_hand",
					ItemMeta: meta,
				},
				Count: 1,
			}

			hand := inv.List("hand")
			if hand == nil {
				inv = append(inv, mt.NamedInvList{
					Name: "hand",
					InvList: mt.InvList{
						Width:  1,
						Stacks: []mt.Stack{handStack},
					},
				})
			} else if len(hand.Stacks) == 0 {
				hand.Width = 1
				hand.Stacks = []mt.Stack{handStack}
			}

			b = &strings.Builder{}
			inv.SerializeKeep(b, sc.inv)
			sc.inv = inv

			sc.client().SendCmd(&mt.ToCltInv{Inv: b.String()})
		case *mt.ToCltAOMsgs:
			for k := range cmd.Msgs {
				sc.swapAOID(&cmd.Msgs[k].ID)
				sc.handleAOMsg(cmd.Msgs[k].Msg)
			}

			sc.client().SendCmd(cmd)
		case *mt.ToCltAORmAdd:
			resp := &mt.ToCltAORmAdd{}

			for _, ao := range cmd.Remove {
				delete(sc.aos, ao)
				resp.Remove = append(resp.Remove, ao)
			}

			for _, ao := range cmd.Add {
				if ao.InitData.Name == sc.client().name {
					sc.client().currentCAO = ao.ID

					if sc.client().playerCAO == 0 {
						sc.client().playerCAO = ao.ID
						for _, msg := range ao.InitData.Msgs {
							sc.handleAOMsg(msg)
						}

						resp.Add = append(resp.Add, ao)
					}
				} else {
					sc.swapAOID(&ao.ID)
					for _, msg := range ao.InitData.Msgs {
						sc.handleAOMsg(msg)
					}

					resp.Add = append(resp.Add, ao)
				}
			}

			sc.client().SendCmd(resp)
		case *mt.ToCltCSMRestrictionFlags:
			cmd.Flags &= ^mt.NoCSMs
			sc.client().SendCmd(cmd)
		case *mt.ToCltDetachedInv:
			var inv mt.Inv
			inv.Deserialize(strings.NewReader(cmd.Inv))

			for k, l := range inv {
				for i := range l.Stacks {
					prepend(sc.name, &inv[k].InvList.Stacks[i].Name)
				}
			}

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

			sc.client().SendCmd(&mt.ToCltDetachedInv{
				Name: cmd.Name,
				Keep: cmd.Keep,
				Len:  cmd.Len,
				Inv:  b.String(),
			})
		case *mt.ToCltMediaPush:
			var exit bool
			for _, f := range sc.client().media {
				if f.name == cmd.Filename {
					exit = true
					break
				}
			}

			if exit {
				break
			}

			prepend(sc.name, &cmd.Filename)
			sc.client().SendCmd(cmd)
		case *mt.ToCltSkyParams:
			for i := range cmd.Textures {
				prependTexture(sc.name, &cmd.Textures[i])
			}
			sc.client().SendCmd(cmd)
		case *mt.ToCltSunParams:
			prependTexture(sc.name, &cmd.Texture)
			prependTexture(sc.name, &cmd.ToneMap)
			prependTexture(sc.name, &cmd.Rise)
			sc.client().SendCmd(cmd)
		case *mt.ToCltMoonParams:
			prependTexture(sc.name, &cmd.Texture)
			prependTexture(sc.name, &cmd.ToneMap)
			sc.client().SendCmd(cmd)
		case *mt.ToCltSetHotbarParam:
			prependTexture(sc.name, &cmd.Img)
			sc.client().SendCmd(cmd)
		case *mt.ToCltUpdatePlayerList:
			if !sc.client().playerListInit {
				sc.client().playerListInit = true
			} else if cmd.Type == mt.InitPlayers {
				cmd.Type = mt.AddPlayers
			}

			sc.client().SendCmd(cmd)
		}
	}
}
