package proxy

import (
	"crypto/subtle"
	"fmt"
	"net"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt"
)

func (cc *ClientConn) process(pkt mt.Pkt) {
	srv := cc.server()

	switch cmd := pkt.Cmd.(type) {
	case *mt.ToSrvNil:
		return
	case *mt.ToSrvInit:
		if cc.state() > csCreated {
			cc.Log("->", "duplicate init")
			return
		}

		cc.setState(csInit)
		if cmd.SerializeVer != latestSerializeVer {
			cc.Log("<-", "invalid serializeVer")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnsupportedVer})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if cmd.MaxProtoVer < latestProtoVer {
			cc.Log("<-", "invalid protoVer")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnsupportedVer})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if len(cmd.PlayerName) == 0 || len(cmd.PlayerName) > maxPlayerNameLen {
			cc.Log("<-", "invalid player name length")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadName})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if !playerNameChars.MatchString(cmd.PlayerName) {
			cc.Log("<-", "invalid player name")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadNameChars})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		cc.name = cmd.PlayerName
		cc.logger.SetPrefix(fmt.Sprintf("[%s %s] ", cc.RemoteAddr(), cc.Name()))

		if authIface.Banned(cc.RemoteAddr().(*net.UDPAddr)) {
			cc.Log("<-", "banned")
			cc.Kick("Banned by proxy.")
			return
		}

		playersMu.Lock()
		_, ok := players[cc.Name()]
		if ok {
			cc.Log("<-", "already connected")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.AlreadyConnected})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			playersMu.Unlock()
			return
		}

		players[cc.Name()] = struct{}{}
		playersMu.Unlock()

		if cc.Name() == "singleplayer" {
			cc.Log("<-", "name is singleplayer")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.BadName})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		// user limit
		if len(players) >= Conf().UserLimit {
			cc.Log("<-", "player limit reached")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.TooManyClts})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
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

		return
	case *mt.ToSrvFirstSRP:
		if cc.state() == csInit {
			if cc.auth.method != mt.FirstSRP {
				cc.Log("->", "unauthorized password change")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			cc.auth = struct {
				method                       mt.AuthMethods
				salt, srpA, srpB, srpM, srpK []byte
			}{}

			if cmd.EmptyPasswd && Conf().RequirePasswd {
				cc.Log("<-", "empty password disallowed")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.EmptyPasswd})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			if err := authIface.SetPasswd(cc.Name(), cmd.Salt, cmd.Verifier); err != nil {
				cc.Log("<-", "set password fail")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.SrvErr})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			cc.Log("->", "set password")
			cc.SendCmd(&mt.ToCltAcceptAuth{
				PlayerPos:       mt.Pos{0, 5, 0},
				MapSeed:         0,
				SendInterval:    Conf().SendInterval,
				SudoAuthMethods: mt.SRP,
			})
		} else {
			if cc.state() < csSudo {
				cc.Log("->", "unauthorized sudo action")
				return
			}

			cc.setState(cc.state() - 1)
			if err := authIface.SetPasswd(cc.Name(), cmd.Salt, cmd.Verifier); err != nil {
				cc.Log("<-", "change password fail")
				cc.SendChatMsg("Password change failed or unavailable.")
				return
			}

			cc.Log("->", "change password")
			cc.SendChatMsg("Password change successful.")
		}

		return
	case *mt.ToSrvSRPBytesA:
		wantSudo := cc.state() == csActive

		if cc.state() != csInit && cc.state() != csActive {
			cc.Log("->", "unexpected authentication")
			return
		}

		if !wantSudo && cc.auth.method != mt.SRP {
			cc.Log("<-", "multiple authentication attempts")
			if wantSudo {
				cc.SendCmd(&mt.ToCltDenySudoMode{})
				return
			}

			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})
			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if !cmd.NoSHA1 {
			cc.Log("<-", "unsupported SHA1 auth")
			return
		}

		cc.auth.method = mt.SRP

		salt, verifier, err := authIface.Passwd(cc.Name())
		if err != nil {
			cc.Log("<-", "SRP data retrieval fail")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.SrvErr})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		cc.auth.salt = salt
		cc.auth.srpA = cmd.A
		cc.auth.srpB, _, cc.auth.srpK, err = srp.Handshake(cc.auth.srpA, verifier)
		if err != nil || cc.auth.srpB == nil {
			cc.Log("<-", "SRP safety check fail")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		cc.SendCmd(&mt.ToCltSRPBytesSaltB{
			Salt: cc.auth.salt,
			B:    cc.auth.srpB,
		})

		return
	case *mt.ToSrvSRPBytesM:
		wantSudo := cc.state() == csActive

		if cc.state() != csInit && cc.state() != csActive {
			cc.Log("->", "unexpected authentication")
			return
		}

		if cc.auth.method != mt.SRP {
			cc.Log("<-", "multiple authentication attempts")
			if wantSudo {
				cc.SendCmd(&mt.ToCltDenySudoMode{})
				return
			}

			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.UnexpectedData})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
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
				cc.Log("<-", "invalid password (sudo)")
				cc.SendCmd(&mt.ToCltDenySudoMode{})
				return
			}

			cc.Log("<-", "invalid password")
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.WrongPasswd})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		return
	case *mt.ToSrvInit2:
		var err error
		cc.itemDefs, cc.aliases, cc.nodeDefs, cc.p0Map, cc.p0SrvMap, cc.media, err = muxContent(cc.Name())
		if err != nil {
			cc.Log("<-", err.Error())
			cc.Kick("Content multiplexing failed.")
			return
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

		return
	case *mt.ToSrvReqMedia:
		cc.sendMedia(cmd.Filenames)
		return
	case *mt.ToSrvCltReady:
		cc.major = cmd.Major
		cc.minor = cmd.Minor
		cc.patch = cmd.Patch
		cc.reservedVer = cmd.Reserved
		cc.versionStr = cmd.Version
		cc.formspecVer = cmd.Formspec

		cc.setState(cc.state() + 1)
		close(cc.initCh)

		return
	case *mt.ToSrvInteract:
		if srv == nil {
			cc.Log("->", "no server")
			return
		}

		if _, ok := cmd.Pointed.(*mt.PointedAO); ok {
			srv.swapAOID(&pkt.Cmd.(*mt.ToSrvInteract).Pointed.(*mt.PointedAO).ID)
		}
	case *mt.ToSrvChatMsg:
		result, isCmd := onChatMsg(cc, cmd)
		if !isCmd {
			break
		} else if result != "" {
			cc.SendChatMsg(result)
		}

		return
	}

	if srv == nil {
		cc.Log("->", "no server")
		return
	}

	srv.Send(pkt)
}
