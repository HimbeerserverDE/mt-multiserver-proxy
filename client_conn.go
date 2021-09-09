package proxy

import (
	"crypto/subtle"
	"errors"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"

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

type ClientConn struct {
	mt.Peer
	srv *ServerConn
	mu  sync.RWMutex

	cstate   clientState
	cstateMu sync.RWMutex
	name     string
	initCh   chan struct{}
	hopMu    sync.Mutex

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

	modChs map[string]struct{}
}

func (cc *ClientConn) Name() string { return cc.Name() }

func (cc *ClientConn) server() *ServerConn {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	return cc.srv
}

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

func (cc *ClientConn) Init() <-chan struct{} { return cc.initCh }

func (cc *ClientConn) Log(dir string, v ...interface{}) {
	if cc.Name() != "" {
		format := "{%s, %s} %s {←|⇶}"
		format += strings.Repeat(" %v", len(v))

		log.Printf(format, append([]interface{}{
			cc.Name(),
			cc.RemoteAddr(),
			dir,
		}, v...)...)
	} else {
		format := "{%s} %s {←|⇶}"
		format += strings.Repeat(" %v", len(v))

		log.Printf(format, append([]interface{}{cc.RemoteAddr(), dir}, v...)...)
	}
}

func handleClt(cc *ClientConn) {
	for {
		pkt, err := cc.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if errors.Is(cc.WhyClosed(), rudp.ErrTimedOut) {
					cc.Log("<->", "timeout")
				} else {
					cc.Log("<->", "disconnect")
				}

				if cc.Name() != "" {
					playersMu.Lock()
					delete(players, cc.Name())
					playersMu.Unlock()
				}

				if cc.server() != nil {
					cc.server().Close()

					cc.server().mu.Lock()
					cc.server().clt = nil
					cc.server().mu.Unlock()

					cc.mu.Lock()
					cc.srv = nil
					cc.mu.Unlock()
				}

				break
			}

			cc.Log("-->", err)
			continue
		}

		switch cmd := pkt.Cmd.(type) {
		case *mt.ToSrvInit:
			if cc.state() > csCreated {
				cc.Log("-->", "duplicate init")
				break
			}

			cc.setState(csInit)
			if cmd.SerializeVer != latestSerializeVer {
				cc.Log("<--", "invalid serializeVer")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnsupportedVer})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if cmd.MaxProtoVer < latestProtoVer {
				cc.Log("<--", "invalid protoVer")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnsupportedVer})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if len(cmd.PlayerName) == 0 || len(cmd.PlayerName) > maxPlayerNameLen {
				cc.Log("<--", "invalid player name length")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadName})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			if ok, _ := regexp.MatchString(playerNameChars, cmd.PlayerName); !ok {
				cc.Log("<--", "invalid player name")
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
			_, ok := players[cc.Name()]
			if ok {
				cc.Log("<--", "already connected")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.AlreadyConnected})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				playersMu.Unlock()
				break
			}

			players[cc.Name()] = struct{}{}
			playersMu.Unlock()

			if cc.Name() == "singleplayer" {
				cc.Log("<--", "name is singleplayer")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadName})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			// user limit
			if len(players) >= Conf().UserLimit {
				cc.Log("<--", "player limit reached")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.TooManyClts})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}

			// reply
			if authIface.Exists(cc.Name()) {
				cc.auth.method = mt.SRP
			} else {
				cc.auth.method = mt.FirstSRP
			}

			cc.SendCmd(&mt.ToCltHello{
				SerializeVer: latestSerializeVer,
				ProtoVer:     latestProtoVer,
				AuthMethods:  cc.auth.method,
				Username:     cc.Name(),
			})
		case *mt.ToSrvFirstSRP:
			if cc.state() == csInit {
				if cc.auth.method != mt.FirstSRP {
					cc.Log("-->", "unauthorized password change")
					ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

					select {
					case <-cc.Closed():
					case <-ack:
						cc.Close()
					}

					break
				}

				cc.auth = struct {
					method                       mt.AuthMethods
					salt, srpA, srpB, srpM, srpK []byte
				}{}

				if cmd.EmptyPasswd && Conf().RequirePasswd {
					cc.Log("<--", "empty password disallowed")
					ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.EmptyPasswd})

					select {
					case <-cc.Closed():
					case <-ack:
						cc.Close()
					}

					break
				}

				if err := authIface.SetPasswd(cc.Name(), cmd.Salt, cmd.Verifier); err != nil {
					cc.Log("<--", "set password fail")
					ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.SrvErr})

					select {
					case <-cc.Closed():
					case <-ack:
						cc.Close()
					}

					break
				}

				cc.Log("-->", "set password")
				cc.SendCmd(&mt.ToCltAcceptAuth{
					PlayerPos:       mt.Pos{0, 5, 0},
					MapSeed:         0,
					SendInterval:    Conf().SendInterval,
					SudoAuthMethods: mt.SRP,
				})
			} else {
				if cc.state() < csSudo {
					cc.Log("-->", "unauthorized sudo action")
					break
				}

				cc.setState(cc.state() - 1)
				if err := authIface.SetPasswd(cc.Name(), cmd.Salt, cmd.Verifier); err != nil {
					cc.Log("<--", "change password fail")
					cc.SendChatMsg("Password change failed or unavailable.")
					break
				}

				cc.Log("-->", "change password")
				cc.SendChatMsg("Password change successful.")
			}
		case *mt.ToSrvSRPBytesA:
			wantSudo := cc.state() == csActive

			if cc.state() != csInit && cc.state() != csActive {
				cc.Log("-->", "unexpected authentication")
				break
			}

			if !wantSudo && cc.auth.method != mt.SRP {
				cc.Log("<--", "multiple authentication attempts")
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
				cc.Log("<--", "unsupported SHA1 auth")
				break
			}

			cc.auth.method = mt.SRP

			salt, verifier, err := authIface.Passwd(cc.Name())
			if err != nil {
				cc.Log("<--", "SRP data retrieval fail")
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
				cc.Log("<--", "SRP safety check fail")
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
			wantSudo := cc.state() == csActive

			if cc.state() != csInit && cc.state() != csActive {
				cc.Log("-->", "unexpected authentication")
				break
			}

			if cc.auth.method != mt.SRP {
				cc.Log("<--", "multiple authentication attempts")
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

			M := srp.ClientProof([]byte(cc.Name()), cc.auth.salt, cc.auth.srpA, cc.auth.srpB, cc.auth.srpK)
			if subtle.ConstantTimeCompare(cmd.M, M) == 1 {
				cc.auth = struct {
					method                       mt.AuthMethods
					salt, srpA, srpB, srpM, srpK []byte
				}{}

				if wantSudo {
					cc.setState(cc.state() + 1)
					cc.SendCmd(&mt.ToCltAcceptSudoMode{})
				} else {
					cc.SendCmd(&mt.ToCltAcceptAuth{
						PlayerPos:       mt.Pos{0, 5, 0},
						MapSeed:         0,
						SendInterval:    Conf().SendInterval,
						SudoAuthMethods: mt.SRP,
					})
				}
			} else {
				if wantSudo {
					cc.Log("<--", "invalid password (sudo)")
					cc.SendCmd(&mt.ToCltDenySudoMode{})
					break
				}

				cc.Log("<--", "invalid password")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.WrongPasswd})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				break
			}
		case *mt.ToSrvInit2:
			cc.itemDefs, cc.aliases, cc.nodeDefs, cc.p0Map, cc.p0SrvMap, cc.media, err = muxContent(cc.Name())
			if err != nil {
				cc.Log("<--", err.Error())

				ack, _ := cc.SendCmd(&mt.ToCltDisco{
					Reason: mt.Custom,
					Custom: "Content multiplexing failed.",
				})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}
			}

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
			if Conf().CSMRF.NoCSMs {
				csmrf |= mt.NoCSMs
			}
			if !Conf().CSMRF.ChatMsgs {
				csmrf |= mt.NoChatMsgs
			}
			if !Conf().CSMRF.ItemDefs {
				csmrf |= mt.NoItemDefs
			}
			if !Conf().CSMRF.NodeDefs {
				csmrf |= mt.NoNodeDefs
			}
			if !Conf().CSMRF.NoLimitMapRange {
				csmrf |= mt.LimitMapRange
			}
			if !Conf().CSMRF.PlayerList {
				csmrf |= mt.NoPlayerList
			}

			cc.SendCmd(&mt.ToCltCSMRestrictionFlags{
				Flags:    csmrf,
				MapRange: Conf().MapRange,
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

			cc.setState(cc.state() + 1)
			close(cc.initCh)
		case *mt.ToSrvInteract:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}

			if _, ok := cmd.Pointed.(*mt.PointedAO); ok {
				cc.server().swapAOID(&cmd.Pointed.(*mt.PointedAO).ID)
			}

			cc.server().SendCmd(cmd)
		case *mt.ToSrvChatMsg:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}

			result, isCmd := onChatMsg(cc, cmd)
			if !isCmd {
				cc.server().SendCmd(cmd)
			} else if result != "" {
				cc.SendChatMsg(result)
			}
		case *mt.ToSrvDeletedBlks:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvFallDmg:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvGotBlks:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvJoinModChan:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvLeaveModChan:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvMsgModChan:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvNodeMetaFields:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvPlayerPos:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvRespawn:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvInvAction:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvInvFields:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		case *mt.ToSrvSelectItem:
			if cc.server() == nil {
				cc.Log("-->", "no server")
				break
			}
			cc.server().SendCmd(cmd)
		}
	}
}
