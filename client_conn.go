package main

import (
	"crypto/subtle"
	"errors"
	"log"
	"net"
	"regexp"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

type clientState uint8

const (
	csCreated clientState = iota
	csInit
	csActive
	csSudo
)

type clientConn struct {
	mt.Peer
	srv *serverConn

	state  clientState
	name   string
	initCh chan struct{}

	auth struct {
		method                       mt.AuthMethods
		salt, srpA, srpB, srpM, srpK []byte
	}

	lang string

	major, minor, patch uint8
	reservedVer         uint8
	versionStr          string
	formspecVer         uint16

	itemDefs []mt.ItemDef
	aliases  []struct{ Alias, Orig string }
	nodeDefs []mt.NodeDef
	p0Map    param0Map
	p0SrvMap param0SrvMap
	media    []mediaFile

	playerCAO, currentCAO mt.AOID

	playerListInit bool
}

func (cc *clientConn) server() *serverConn { return cc.srv }

func (cc *clientConn) init() <-chan struct{} { return cc.initCh }

func (cc *clientConn) log(dir, msg string) {
	if cc.name != "" {
		log.Printf("{%s, %s} %s {←|⇶} %s", cc.name, cc.RemoteAddr(), dir, msg)
	} else {
		log.Printf("{%s} %s {←|⇶} %s", cc.RemoteAddr(), dir, msg)
	}
}

func handleClt(cc *clientConn) {
	for {
		pkt, err := cc.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if errors.Is(cc.WhyClosed(), rudp.ErrTimedOut) {
					cc.log("<->", "timeout")
				} else {
					cc.log("<->", "disconnect")
				}

				if cc.server() != nil {
					cc.server().Close()
					cc.server().clt = nil
					cc.srv = nil
				}

				if cc.name != "" {
					playersMu.Lock()
					delete(players, cc.name)
					playersMu.Unlock()
				}

				break
			}

			cc.log("-->", err.Error())
			continue
		}

		switch cmd := pkt.Cmd.(type) {
		case *mt.ToSrvInit:
			if cc.state > csCreated {
				cc.log("-->", "duplicate init")
				break
			}

			cc.state = csInit

			if cmd.SerializeVer != latestSerializeVer {
				cc.log("<--", "invalid serializeVer")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnsupportedVer})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if cmd.MaxProtoVer < latestProtoVer {
				cc.log("<--", "invalid protoVer")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnsupportedVer})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if len(cmd.PlayerName) == 0 || len(cmd.PlayerName) > maxPlayerNameLen {
				cc.log("<--", "invalid player name length")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadName})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if ok, _ := regexp.MatchString(playerNameChars, cmd.PlayerName); !ok {
				cc.log("<--", "invalid player name")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadNameChars})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			cc.name = cmd.PlayerName

			playersMu.Lock()
			_, ok := players[cc.name]
			if ok {
				cc.log("<--", "already connected")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.AlreadyConnected})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				playersMu.Unlock()
				break
			}

			players[cc.name] = struct{}{}
			playersMu.Unlock()

			if cc.name == "singleplayer" {
				cc.log("<--", "name is singleplayer")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadName})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			// user limit
			if len(players) >= conf.UserLimit {
				cc.log("<--", "player limit reached")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.TooManyClts})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			// reply
			if authIface.Exists(cc.name) {
				cc.auth.method = mt.SRP
			} else {
				cc.auth.method = mt.FirstSRP
			}

			cc.SendCmd(&mt.ToCltHello{
				SerializeVer: latestSerializeVer,
				ProtoVer:     latestProtoVer,
				AuthMethods:  cc.auth.method,
				Username:     cc.name,
			})
		case *mt.ToSrvFirstSRP:
			if cc.state == csInit {
				if cc.auth.method != mt.FirstSRP {
					cc.log("-->", "unauthorized password change")
					ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

					select {
					case <-cc.Closed():
					case <-ack:
						cc.Close()
					}

					break
				}

				cc.auth.method = 0

				if cmd.EmptyPasswd && conf.RequirePasswd {
					cc.log("<--", "empty password disallowed")
					ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.EmptyPasswd})

					select {
					case <-cc.Closed():
					case <-ack:
						cc.Close()
					}

					break
				}

				if err := authIface.SetPasswd(cc.name, cmd.Salt, cmd.Verifier); err != nil {
					cc.log("<--", "set password fail")
					ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.SrvErr})

					select {
					case <-cc.Closed():
					case <-ack:
						cc.Close()
					}

					break
				}

				cc.log("-->", "set password")
				cc.SendCmd(&mt.ToCltAcceptAuth{
					PlayerPos:       mt.Pos{0, 5, 0},
					MapSeed:         0,
					SendInterval:    conf.SendInterval,
					SudoAuthMethods: mt.SRP,
				})
			} else {
				if cc.state < csSudo {
					cc.log("-->", "unauthorized sudo action")
					break
				}

				cc.state--

				if err := authIface.SetPasswd(cc.name, cmd.Salt, cmd.Verifier); err != nil {
					cc.log("<--", "change password fail")
					cc.SendCmd(&mt.ToCltChatMsg{
						Type:      mt.SysMsg,
						Text:      "Password change failed or unavailable.",
						Timestamp: time.Now().Unix(),
					})
					break
				}

				cc.log("-->", "change password")
				cc.SendCmd(&mt.ToCltChatMsg{
					Type:      mt.SysMsg,
					Text:      "Password change successful.",
					Timestamp: time.Now().Unix(),
				})
			}
		case *mt.ToSrvSRPBytesA:
			wantSudo := cc.state == csActive

			if cc.state != csInit && cc.state != csActive {
				cc.log("-->", "unexpected authentication")
				break
			}

			if !wantSudo && cc.auth.method != mt.SRP {
				cc.log("<--", "multiple authentication attempts")
				if wantSudo {
					cc.SendCmd(&mt.ToCltDenySudoMode{})
					break
				}

				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if !cmd.NoSHA1 {
				cc.log("<--", "unsupported SHA1 auth")
				break
			}

			cc.auth.method = mt.SRP

			salt, verifier, err := authIface.Passwd(cc.name)
			if err != nil {
				cc.log("<--", "SRP data retrieval fail")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.SrvErr})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			cc.auth.salt = salt
			cc.auth.srpA = cmd.A
			cc.auth.srpB, _, cc.auth.srpK, err = srp.Handshake(cc.auth.srpA, verifier)
			if err != nil || cc.auth.srpB == nil {
				cc.log("<--", "SRP safety check fail")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			cc.SendCmd(&mt.ToCltSRPBytesSaltB{
				Salt: cc.auth.salt,
				B:    cc.auth.srpB,
			})
		case *mt.ToSrvSRPBytesM:
			wantSudo := cc.state == csActive

			if cc.state != csInit && cc.state != csActive {
				cc.log("-->", "unexpected authentication")
				break
			}

			if cc.auth.method != mt.SRP {
				cc.log("<--", "multiple authentication attempts")
				if wantSudo {
					cc.SendCmd(&mt.ToCltDenySudoMode{})
					break
				}

				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			M := srp.ClientProof([]byte(cc.name), cc.auth.salt, cc.auth.srpA, cc.auth.srpB, cc.auth.srpK)
			if subtle.ConstantTimeCompare(cmd.M, M) == 1 {
				cc.auth.method = 0

				if wantSudo {
					cc.state++
					cc.SendCmd(&mt.ToCltAcceptSudoMode{})
				} else {
					cc.SendCmd(&mt.ToCltAcceptAuth{
						PlayerPos:       mt.Pos{0, 5, 0},
						MapSeed:         0,
						SendInterval:    conf.SendInterval,
						SudoAuthMethods: mt.SRP,
					})
				}
			} else {
				if wantSudo {
					cc.log("<--", "invalid password (sudo)")
					cc.SendCmd(&mt.ToCltDenySudoMode{})
					break
				}

				cc.log("<--", "invalid password")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.WrongPasswd})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}
		case *mt.ToSrvInit2:
			cc.itemDefs, cc.aliases, cc.nodeDefs, cc.p0Map, cc.p0SrvMap, cc.media, err = muxContent(cc.name)
			cc.SendCmd(&mt.ToCltItemDefs{
				Defs:    cc.itemDefs,
				Aliases: cc.aliases,
			})
			cc.SendCmd(&mt.ToCltNodeDefs{Defs: cc.nodeDefs})

			cc.itemDefs = []mt.ItemDef{}
			cc.nodeDefs = []mt.NodeDef{}

			var files []struct{ Name, Base64SHA1 string }
			for _, f := range cc.media {
				files = append(files, struct{ Name, Base64SHA1 string }{
					Name:       f.name,
					Base64SHA1: f.base64SHA1,
				})
			}

			cc.SendCmd(&mt.ToCltAnnounceMedia{Files: files})
			cc.lang = cmd.Lang

			var csmrf mt.CSMRestrictionFlags
			if conf.CSMRF.NoCSMs {
				csmrf |= mt.NoCSMs
			}
			if !conf.CSMRF.ChatMsgs {
				csmrf |= mt.NoChatMsgs
			}
			if !conf.CSMRF.ItemDefs {
				csmrf |= mt.NoItemDefs
			}
			if !conf.CSMRF.NodeDefs {
				csmrf |= mt.NoNodeDefs
			}
			if !conf.CSMRF.NoLimitMapRange {
				csmrf |= mt.LimitMapRange
			}
			if !conf.CSMRF.PlayerList {
				csmrf |= mt.NoPlayerList
			}

			cc.SendCmd(&mt.ToCltCSMRestrictionFlags{
				Flags:    csmrf,
				MapRange: conf.MapRange,
			})
		case *mt.ToSrvReqMedia:
			cc.sendMedia(cmd.Filenames)
		case *mt.ToSrvCltReady:
			cc.major = cmd.Major
			cc.minor = cmd.Minor
			cc.patch = cmd.Patch
			cc.reservedVer = cmd.Reserved
			cc.versionStr = cmd.Version
			cc.formspecVer = cmd.Formspec

			cc.state++
			close(cc.initCh)
		case *mt.ToSrvInteract:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}

			if _, ok := cmd.Pointed.(*mt.PointedAO); ok {
				cc.server().swapAOID(&cmd.Pointed.(*mt.PointedAO).ID)
			}

			cc.server().SendCmd(cmd)
		case *mt.ToSrvChatMsg:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvDeletedBlks:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvFallDmg:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvGotBlks:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvJoinModChan:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvLeaveModChan:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvMsgModChan:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvNodeMetaFields:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvPlayerPos:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvRespawn:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvSelectItem:
			if cc.server() == nil {
				cc.log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		}
	}
}
