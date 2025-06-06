package proxy

import (
	"crypto/subtle"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/HimbeerserverDE/mt"
	"github.com/HimbeerserverDE/srp"
)

func (cc *ClientConn) process(pkt mt.Pkt) {
	defer func() {
		cltLeaveModChans(cc)
	}()

	srv := cc.server()

	forward := func(pkt mt.Pkt) {
		if srv == nil {
			cc.Log("->", "no server")
			return
		}

		srv.Send(pkt)
	}

	switch cmd := pkt.Cmd.(type) {
	case *mt.ToSrvNil:
		return
	case *mt.ToSrvInit:
		if cc.state() > csCreated {
			cc.Log("->", "duplicate init")
			return
		}

		cc.setState(csInit)
		if cmd.SerializeVer != serializeVer {
			cc.Log("<-", "unsupported serializeVer", cmd.SerializeVer, "expect", serializeVer)
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.UnsupportedVer})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if cmd.MaxProtoVer != protoVer || cmd.MinProtoVer > protoVer {
			cc.Log("<-", "unsupported protoVer range min", cmd.MinProtoVer, "max", cmd.MaxProtoVer, "expect", protoVer)
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.UnsupportedVer})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if len(cmd.PlayerName) == 0 || len(cmd.PlayerName) > maxPlayerNameLen {
			cc.Log("<-", "invalid player name length")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.BadName})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		cc.name = cmd.PlayerName
		cc.logger.SetPrefix(fmt.Sprintf("[%s %s] ", cc.RemoteAddr(), cc.Name()))

		playersMu.Lock()
		_, ok := players[cc.Name()]
		if ok {
			cc.Log("<-", "already connected")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.AlreadyConnected})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			// Needed so that the username doesn't get removed from
			// the player list which would allow other clients to
			// bypass this check.
			cc.name = ""

			playersMu.Unlock()
			return
		}

		players[cc.Name()] = struct{}{}
		playersMu.Unlock()

		if !playerNameChars.MatchString(cmd.PlayerName) {
			cc.Log("<-", "invalid player name")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.BadNameChars})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		if cc.Name() == "singleplayer" {
			cc.Log("<-", "name is singleplayer")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.BadName})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		// user limit
		if len(players) > Conf().UserLimit {
			cc.Log("<-", "player limit reached")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.TooManyClts})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		ip := cc.RemoteAddr().(*net.UDPAddr).IP.String()
		if DefaultAuth().Banned(ip, cc.Name()) {
			cc.Log("<-", "banned")
			cc.Kick("Banned by proxy.")
			return
		}

		// reply
		if DefaultAuth().Exists(cc.Name()) {
			cc.auth.method = mt.SRP
		} else {
			cc.auth.method = mt.FirstSRP
		}

		cc.SendCmd(&mt.ToCltHello{
			SerializeVer: serializeVer,
			ProtoVer:     protoVer,
			AuthMethods:  cc.auth.method,
			Username:     cc.Name(),
		})

		return
	case *mt.ToSrvFirstSRP:
		if cc.state() == csInit {
			if cc.auth.method != mt.FirstSRP {
				cc.Log("->", "unauthorized password change")
				ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.UnexpectedData})

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
				ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.EmptyPasswd})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			if err := DefaultAuth().SetPasswd(cc.Name(), cmd.Salt, cmd.Verifier); err != nil {
				cc.Log("<-", "set password fail")
				ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.SrvErr})

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
			cc.new = true
		} else {
			if cc.state() < csSudo {
				cc.Log("->", "unauthorized sudo action")
				return
			}

			cc.setState(csActive)
			if err := DefaultAuth().SetPasswd(cc.Name(), cmd.Salt, cmd.Verifier); err != nil {
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

			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.UnexpectedData})
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

		salt, verifier, err := DefaultAuth().Passwd(cc.Name())
		if err != nil {
			cc.Log("<-", "SRP data retrieval fail")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.SrvErr})

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
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.UnexpectedData})

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

			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.UnexpectedData})

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
				cc.setState(csSudo)
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
			ip := cc.RemoteAddr().(*net.UDPAddr).IP.String()
			if err := DefaultAuth().RecordFail(ip, cc.Name(), wantSudo); err != nil {
				cc.Log("<-", "record auth fail:", err)
			}

			if wantSudo {
				cc.Log("<-", "invalid password (sudo)")
				cc.SendCmd(&mt.ToCltDenySudoMode{})
				return
			}

			cc.Log("<-", "invalid password")
			ack, _ := cc.SendCmd(&mt.ToCltKick{Reason: mt.WrongPasswd})

			select {
			case <-cc.Closed():
			case <-ack:
				cc.Close()
			}

			return
		}

		return
	case *mt.ToSrvInit2:
		var remotes []string
		var err error
		cc.denyPools, cc.itemDefs, cc.aliases, cc.nodeDefs, cc.p0Map, cc.p0SrvMap, cc.media, remotes, err = muxContent(cc.Name())
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

		cc.itemDefs = nil
		cc.aliases = nil
		cc.nodeDefs = nil

		var files []struct{ Name, Base64SHA1 string }
		for _, f := range cc.media {
			files = append(files, struct{ Name, Base64SHA1 string }{
				Name:       f.name,
				Base64SHA1: f.base64SHA1,
			})
		}

		cc.SendCmd(&mt.ToCltAnnounceMedia{
			Files: files,
			URL:   strings.Join(remotes, ","),
		})
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
		// Don't leak media memory, regardless of whether the client
		// requested anything.
		cc.media = nil

		cc.major = cmd.Major
		cc.minor = cmd.Minor
		cc.patch = cmd.Patch
		cc.reservedVer = cmd.Reserved
		cc.versionStr = cmd.Version
		cc.formspecVer = cmd.Formspec

		cc.setState(csActive)
		handleJoin(cc)
		close(cc.initCh)

		return
	case *mt.ToSrvInteract:
		if srv == nil {
			cc.Log("->", "no server")
			return
		}

		if _, ok := cmd.Pointed.(*mt.PointedAO); ok {
			srv.swapAOID(&cmd.Pointed.(*mt.PointedAO).ID)
		}

		if handleInteraction(cmd, cc) { // if return true: already handled
			return
		}
	case *mt.ToSrvChatMsg:
		done := make(chan struct{})

		go func(done chan<- struct{}) {
			result, isCmd := onChatMsg(cc, cmd)
			if !isCmd {
				cmd.Msg = result
				forward(pkt)
			} else if result != "" {
				cc.SendChatMsg(result)
			}

			close(done)
		}(done)

		go func(done <-chan struct{}) {
			select {
			case <-done:
			case <-time.After(ChatCmdTimeout):
				cc.Log("<-", "timeout chat msg", cmd.Msg)
				cc.SendChatMsg("Proxy chat message processing timeout.")
			}
		}(done)

		return
	case *mt.ToSrvCltInfo:
		// Store for any future hops (need to send it to the new server).
		cc.cltInfo = cmd
	case *mt.ToSrvInvAction:
		onInvActionMu.RLock()
		defer onInvActionMu.RUnlock()

		for _, handler := range onInvAction {
			cmd.Action = handler(cc, cmd.Action)
		}

		if cmd.Action == "" {
			return
		}
	case *mt.ToSrvInvFields:
		if handleOnPlayerReceiveFields(cc, cmd) {
			return
		}
	case *mt.ToSrvJoinModChan:
		modChanSubscriberMu.Lock()
		defer modChanSubscriberMu.Unlock()

		subs, _ := modChanSubscribers[cmd.Channel]
		modChanSubscribers[cmd.Channel] = append(subs, cc)
	case *mt.ToSrvLeaveModChan:
		cltLeaveModChan(cc, cmd.Channel)
	case *mt.ToSrvMsgModChan:
		if handleCltModChanMsg(cc, cmd) {
			return
		}
	}

	forward(pkt)
}

func (sc *ServerConn) process(pkt mt.Pkt) {
	clt := sc.client()
	if clt == nil {
		sc.Log("<-", "no client")
		return
	}

	switch cmd := pkt.Cmd.(type) {
	case *mt.ToCltHello:
		if sc.auth.method != 0 {
			sc.Log("<-", "unexpected authentication")
			sc.Close()
			return
		}

		sc.setState(csActive)
		if cmd.AuthMethods&mt.FirstSRP != 0 {
			sc.auth.method = mt.FirstSRP
		} else {
			sc.auth.method = mt.SRP
		}

		if cmd.SerializeVer != serializeVer {
			sc.Log("<-", "invalid serializeVer")
			return
		}

		switch sc.auth.method {
		case mt.SRP:
			var err error
			sc.auth.srpA, sc.auth.a, err = srp.InitiateHandshake()
			if err != nil {
				sc.Log("->", err)
				return
			}

			sc.SendCmd(&mt.ToSrvSRPBytesA{
				A:      sc.auth.srpA,
				NoSHA1: true,
			})
		case mt.FirstSRP:
			id := strings.ToLower(clt.Name())

			salt, verifier, err := srp.NewClient([]byte(id), []byte{})
			if err != nil {
				sc.Log("->", err)
				return
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

		return
	case *mt.ToCltSRPBytesSaltB:
		if sc.auth.method != mt.SRP {
			sc.Log("<-", "multiple authentication attempts")
			return
		}

		id := strings.ToLower(clt.Name())

		var err error
		sc.auth.srpK, err = srp.CompleteHandshake(sc.auth.srpA, sc.auth.a, []byte(id), []byte{}, cmd.Salt, cmd.B)
		if err != nil {
			sc.Log("->", err)
			return
		}

		M := srp.ClientProof([]byte(clt.Name()), cmd.Salt, sc.auth.srpA, cmd.B, sc.auth.srpK)
		if M == nil {
			sc.Log("<-", "SRP safety check fail")
			return
		}

		sc.SendCmd(&mt.ToSrvSRPBytesM{
			M: M,
		})

		return
	case *mt.ToCltKick:
		sc.Log("<-", "deny access", cmd)

		if cmd.Reason == mt.Shutdown || cmd.Reason == mt.Crash || cmd.Reason == mt.SrvErr || cmd.Reason == mt.TooManyClts || cmd.Reason == mt.UnsupportedVer {
			clt.SendChatMsg("A kick occured, switching to fallback server. Reason:", cmd)

			clt.whyKicked = cmd

			clt.fallback()
			return
		}

		ack, _ := clt.SendCmd(cmd)

		select {
		case <-clt.Closed():
		case <-ack:
			clt.Close()

			sc.mu.Lock()
			sc.clt = nil
			sc.mu.Unlock()
		}

		return
	case *mt.ToCltAcceptAuth:
		sc.auth = struct {
			method              mt.AuthMethods
			salt, srpA, a, srpK []byte
		}{}
		sc.SendCmd(&mt.ToSrvInit2{Lang: clt.lang})

		return
	case *mt.ToCltDenySudoMode:
		sc.Log("<-", "deny sudo")
		return
	case *mt.ToCltAcceptSudoMode:
		sc.Log("<-", "accept sudo")
		sc.setState(csSudo)
		return
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
		sc.setState(csActive)
		close(sc.initCh)

		return
	case *mt.ToCltMedia:
		tokens := make([]uint32, 0, len(cmd.Files))
		for i, f := range cmd.Files {
			prepend(sc.mediaPool, &cmd.Files[i].Name)

			dynInfo, ok := sc.dynMedia[f.Name]
			if ok {
				if !dynInfo.ephemeral {
					cacheMedia(f.Data)
				}

				tokens = append(tokens, dynInfo.token)
				delete(sc.dynMedia, f.Name)
			}
		}

		sc.SendCmd(&mt.ToSrvHaveMedia{Tokens: tokens})
	case *mt.ToCltItemDefs:
		return
	case *mt.ToCltNodeDefs:
		return
	case *mt.ToCltInv:
		var oldInv mt.Inv
		oldB := &strings.Builder{}
		sc.inv.Serialize(oldB)
		oldInv.Deserialize(strings.NewReader(oldB.String()))
		sc.inv.Deserialize(strings.NewReader(cmd.Inv))
		sc.prependInvKeep(sc.inv, oldInv)

		handStack := mt.Stack{
			Item: mt.Item{
				Name: sc.mediaPool + "_hand",
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
		return
	case *mt.ToCltAOMsgs:
		for k := range cmd.Msgs {
			sc.swapAOID(&cmd.Msgs[k].ID)
			sc.handleAOMsg(cmd.Msgs[k].Msg)
		}
	case *mt.ToCltAORmAdd:
		resp := &mt.ToCltAORmAdd{}

		for _, ao := range cmd.Remove {
			delete(sc.aos, ao)
			resp.Remove = append(resp.Remove, ao)
		}

		for _, ao := range cmd.Add {
			if ao.InitData.Name == clt.name {
				clt.currentCAO = ao.ID

				if !clt.hasPlayerCAO() {
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
		return
	case *mt.ToCltCSMRestrictionFlags:
		if Conf().DropCSMRF {
			return
		}

		cmd.Flags &= ^mt.NoCSMs
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

		return
	case *mt.ToCltMediaPush:
		filename := cmd.Filename
		prepend(sc.mediaPool, &cmd.Filename)

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

		sc.dynMedia[cmd.Filename] = struct {
			ephemeral bool
			token     uint32
		}{
			ephemeral: cmd.Ephemeral,
			token:     cmd.CallbackToken,
		}

		sc.SendCmd(&mt.ToSrvReqMedia{Filenames: []string{filename}})
	case *mt.ToCltSkyParams:
		for i := range cmd.Textures {
			prependTexture(sc.mediaPool, &cmd.Textures[i])
		}
	case *mt.ToCltSunParams:
		prependTexture(sc.mediaPool, &cmd.Texture)
		prependTexture(sc.mediaPool, &cmd.ToneMap)
		prependTexture(sc.mediaPool, &cmd.Rise)
	case *mt.ToCltMoonParams:
		prependTexture(sc.mediaPool, &cmd.Texture)
		prependTexture(sc.mediaPool, &cmd.ToneMap)
	case *mt.ToCltSetHotbarParam:
		prependTexture(sc.mediaPool, &cmd.Img)
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
	case *mt.ToCltSpawnParticle:
		prependTexture(sc.mediaPool, &cmd.TextureName)
		sc.globalParam0(&cmd.NodeParam0)
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
	case *mt.ToCltAddNode:
		sc.globalParam0(&cmd.Node.Param0)
	case *mt.ToCltAddParticleSpawner:
		prependTexture(sc.mediaPool, &cmd.TextureName)
		sc.swapAOID(&cmd.AttachedAOID)
		sc.globalParam0(&cmd.NodeParam0)
		sc.particleSpawners[cmd.ID] = struct{}{}
	case *mt.ToCltDelParticleSpawner:
		delete(sc.particleSpawners, cmd.ID)
	case *mt.ToCltPlaySound:
		prepend(sc.mediaPool, &cmd.Name)
		sc.swapAOID(&cmd.SrcAOID)
		if cmd.Loop {
			sc.sounds[cmd.ID] = struct{}{}
		}
	case *mt.ToCltFadeSound:
		delete(sc.sounds, cmd.ID)
	case *mt.ToCltStopSound:
		delete(sc.sounds, cmd.ID)
	case *mt.ToCltAddHUD:
		sc.prependHUD(cmd.Type, cmd)

		sc.huds[cmd.ID] = cmd.Type
	case *mt.ToCltChangeHUD:
		sc.prependHUD(sc.huds[cmd.ID], cmd)
	case *mt.ToCltRmHUD:
		delete(sc.huds, cmd.ID)
	case *mt.ToCltShowFormspec:
		sc.prependFormspec(&cmd.Formspec)
	case *mt.ToCltFormspecPrepend:
		sc.prependFormspec(&cmd.Prepend)
	case *mt.ToCltInvFormspec:
		sc.prependFormspec(&cmd.Formspec)
	case *mt.ToCltMinimapModes:
		for i := range cmd.Modes {
			prependTexture(sc.mediaPool, &cmd.Modes[i].Texture)
		}
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
	case *mt.ToCltModChanMsg:
		if handleSrvModChanMsg(clt, cmd) {
			return
		}
	case *mt.ToCltModChanSig:
		reportStatus := func(ch chan bool, status bool) {
			ch <- status
			delete(sc.modChanJoinChs[cmd.Channel], ch)
		}

		switch cmd.Signal {
		case mt.JoinOK:
			sc.modChanJoinChMu.Lock()
			defer sc.modChanJoinChMu.Unlock()

			for ch := range sc.modChanJoinChs[cmd.Channel] {
				go reportStatus(ch, true)
			}

			if _, ok := clt.modChs[cmd.Channel]; ok {
				return
			}
			clt.modChs[cmd.Channel] = struct{}{}
		case mt.JoinFail:
			sc.modChanJoinChMu.Lock()
			defer sc.modChanJoinChMu.Unlock()

			for ch := range sc.modChanJoinChs[cmd.Channel] {
				go reportStatus(ch, false)
			}

			fallthrough
		case mt.LeaveOK:
			sc.modChanLeaveChMu.Lock()
			defer sc.modChanLeaveChMu.Unlock()

			for ch := range sc.modChanLeaveChs[cmd.Channel] {
				go reportStatus(ch, true)
			}

			delete(clt.modChs, cmd.Channel)
		case mt.LeaveFail:
			sc.modChanLeaveChMu.Lock()
			defer sc.modChanLeaveChMu.Unlock()

			for ch := range sc.modChanLeaveChs[cmd.Channel] {
				go reportStatus(ch, false)
			}
		}
	}

	clt.Send(pkt)
}
