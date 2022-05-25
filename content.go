package proxy

import (
	"crypto/sha1"
	"embed"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

var disallowedChars = regexp.MustCompile("[^a-zA-Z0-9-_.:]")

var b64 = base64.StdEncoding

//go:embed textures/*
var textures embed.FS

type mediaFile struct {
	name       string
	base64SHA1 string
	data       []byte
}

type contentConn struct {
	mt.Peer

	logger *log.Logger

	cstate         clientState
	cstateMu       sync.RWMutex
	name, userName string
	doneCh         chan struct{}

	auth struct {
		method              mt.AuthMethods
		salt, srpA, a, srpK []byte
	}

	mediaPool string

	itemDefs []mt.ItemDef
	aliases  []struct{ Alias, Orig string }

	nodeDefs []mt.NodeDef

	media   []mediaFile
	remotes []string
}

func (cc *contentConn) state() clientState {
	cc.cstateMu.RLock()
	defer cc.cstateMu.RUnlock()

	return cc.cstate
}

func (cc *contentConn) setState(state clientState) {
	cc.cstateMu.Lock()
	defer cc.cstateMu.Unlock()

	cc.cstate = state
}

func (cc *contentConn) done() <-chan struct{} { return cc.doneCh }

func (cc *contentConn) addDefaultTextures() error {
	dir, err := textures.ReadDir("textures")
	if err != nil {
		return err
	}

	cc.media = make([]mediaFile, 0, len(dir))
	for _, f := range dir {
		data, err := textures.ReadFile("textures/" + f.Name())
		if err != nil {
			return err
		}

		sum := sha1.Sum(data)
		cc.media = append(cc.media, mediaFile{
			name:       f.Name(),
			base64SHA1: b64.EncodeToString(sum[:]),
			data:       data,
		})
	}

	return nil
}

func (cc *contentConn) log(dir string, v ...interface{}) {
	cc.logger.Println(append([]interface{}{dir}, v...)...)
}

func handleContent(cc *contentConn) {
	defer close(cc.doneCh)

	go func() {
		init := make(chan struct{})
		defer close(init)

		go func(init <-chan struct{}) {
			select {
			case <-init:
			case <-time.After(10 * time.Second):
				cc.log("->", "timeout")
				cc.Close()
			}
		}(init)

		for cc.state() == csCreated {
			cc.SendCmd(&mt.ToSrvInit{
				SerializeVer: serializeVer,
				MinProtoVer:  protoVer,
				MaxProtoVer:  protoVer,
				PlayerName:   cc.userName,
			})
			time.Sleep(500 * time.Millisecond)
		}
	}()

	for {
		pkt, err := cc.Recv()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				if errors.Is(cc.WhyClosed(), rudp.ErrTimedOut) {
					cc.log("<->", "timeout")
				}

				cc.setState(csInit)
				break
			}

			cc.log("<-", err)
			continue
		}

		switch cmd := pkt.Cmd.(type) {
		case *mt.ToCltHello:
			if cc.auth.method != 0 {
				cc.log("<-", "unexpected authentication")
				cc.Close()
				break
			}

			cc.setState(csActive)
			if cmd.AuthMethods&mt.FirstSRP != 0 {
				cc.auth.method = mt.FirstSRP
			} else {
				cc.auth.method = mt.SRP
			}

			if cmd.SerializeVer != serializeVer {
				cc.log("<-", "invalid serializeVer")
				break
			}

			switch cc.auth.method {
			case mt.SRP:
				cc.auth.srpA, cc.auth.a, err = srp.InitiateHandshake()
				if err != nil {
					cc.log("->", err)
					break
				}

				cc.SendCmd(&mt.ToSrvSRPBytesA{
					A:      cc.auth.srpA,
					NoSHA1: true,
				})
			case mt.FirstSRP:
				id := strings.ToLower(cc.userName)

				salt, verifier, err := srp.NewClient([]byte(id), []byte{})
				if err != nil {
					cc.log("->", err)
					break
				}

				cc.SendCmd(&mt.ToSrvFirstSRP{
					Salt:        salt,
					Verifier:    verifier,
					EmptyPasswd: true,
				})
			default:
				cc.log("<->", "invalid auth method")
				cc.Close()
			}
		case *mt.ToCltSRPBytesSaltB:
			if cc.auth.method != mt.SRP {
				cc.log("<-", "multiple authentication attempts")
				break
			}

			id := strings.ToLower(cc.userName)

			cc.auth.srpK, err = srp.CompleteHandshake(cc.auth.srpA, cc.auth.a, []byte(id), []byte{}, cmd.Salt, cmd.B)
			if err != nil {
				cc.log("->", err)
				break
			}

			M := srp.ClientProof([]byte(cc.userName), cmd.Salt, cc.auth.srpA, cmd.B, cc.auth.srpK)
			if M == nil {
				cc.log("<-", "SRP safety check fail")
				break
			}

			cc.SendCmd(&mt.ToSrvSRPBytesM{
				M: M,
			})
		case *mt.ToCltKick:
			cc.log("<-", "deny access", cmd)
		case *mt.ToCltAcceptAuth:
			cc.auth.method = 0
			cc.SendCmd(&mt.ToSrvInit2{})
		case *mt.ToCltItemDefs:
			for _, def := range cmd.Defs {
				cc.itemDefs = append(cc.itemDefs, def)
			}
			cc.aliases = cmd.Aliases
		case *mt.ToCltNodeDefs:
			for _, def := range cmd.Defs {
				cc.nodeDefs = append(cc.nodeDefs, def)
			}
		case *mt.ToCltAnnounceMedia:
			var filenames []string

			for _, f := range cmd.Files {

				if cc.fromCache(f.Name, f.Base64SHA1) {
					continue
				}

				filenames = append(filenames, f.Name)

				for i, mf := range cc.media {
					if mf.name == f.Name {
						cc.media[i].base64SHA1 = f.Base64SHA1
						continue
					}
				}

				cc.media = append(cc.media, mediaFile{
					name:       f.Name,
					base64SHA1: f.Base64SHA1,
				})
			}

			cc.remotes = strings.Split(cmd.URL, ",")
			for k, v := range cc.remotes {
				cc.remotes[k] = strings.TrimSpace(v)
			}

			cc.SendCmd(&mt.ToSrvReqMedia{Filenames: filenames})
		case *mt.ToCltMedia:
			for _, f := range cmd.Files {
				for i, af := range cc.media {
					if af.name == f.Name {
						cc.media[i].data = f.Data
						break
					}
				}
			}

			if cmd.I == cmd.N-1 {
				cc.updateCache()
				cc.Close()
			}
		}
	}
}

func (cc *ClientConn) sendMedia(filenames []string) {
	var bunches [][]struct {
		Name string
		Data []byte
	}
	bunches = append(bunches, []struct {
		Name string
		Data []byte
	}{})

	var bunchSize int
	for _, filename := range filenames {
		var known bool
		for _, f := range cc.media {
			if f.name == filename {
				mfile := struct {
					Name string
					Data []byte
				}{
					Name: f.name,
					Data: f.data,
				}
				bunches[len(bunches)-1] = append(bunches[len(bunches)-1], mfile)

				bunchSize += len(f.data)
				if bunchSize >= bytesPerMediaBunch {
					bunches = append(bunches, []struct {
						Name string
						Data []byte
					}{})
					bunchSize = 0
				}

				known = true
				break
			}
		}

		if !known {
			cc.Log("->", "request unknown media file")
			continue
		}
	}

	for i := uint16(0); i < uint16(len(bunches)); i++ {
		cc.SendCmd(&mt.ToCltMedia{
			N:     uint16(len(bunches)),
			I:     i,
			Files: bunches[i],
		})
	}
}

type param0Map map[string]map[mt.Content]mt.Content
type param0SrvMap map[mt.Content]struct {
	name   string
	param0 mt.Content
}

func muxItemDefs(conns []*contentConn) ([]mt.ItemDef, []struct{ Alias, Orig string }) {
	var itemDefs []mt.ItemDef
	var aliases []struct{ Alias, Orig string }

	itemDefs = append(itemDefs, mt.ItemDef{
		Type:       mt.ToolItem,
		InvImg:     "wieldhand.png",
		WieldScale: [3]float32{1, 1, 1},
		StackMax:   1,
		ToolCaps: mt.ToolCaps{
			NonNil: true,
		},
		PointRange: 4,
	})

	for _, cc := range conns {
		<-cc.done()
		for _, def := range cc.itemDefs {
			if def.Name == "" {
				def.Name = "hand"
			}

			prepend(cc.mediaPool, &def.Name)
			prependTexture(cc.mediaPool, &def.InvImg)
			prependTexture(cc.mediaPool, &def.WieldImg)
			prepend(cc.mediaPool, &def.PlacePredict)
			prepend(cc.mediaPool, &def.PlaceSnd.Name)
			prepend(cc.mediaPool, &def.PlaceFailSnd.Name)
			prependTexture(cc.mediaPool, &def.Palette)
			prependTexture(cc.mediaPool, &def.InvOverlay)
			prependTexture(cc.mediaPool, &def.WieldOverlay)
			itemDefs = append(itemDefs, def)
		}

		for _, alias := range cc.aliases {
			prepend(cc.mediaPool, &alias.Alias)
			prepend(cc.mediaPool, &alias.Orig)

			aliases = append(aliases, struct{ Alias, Orig string }{
				Alias: alias.Alias,
				Orig:  alias.Orig,
			})
		}
	}

	return itemDefs, aliases
}

func muxNodeDefs(conns []*contentConn) (nodeDefs []mt.NodeDef, p0Map param0Map, p0SrvMap param0SrvMap) {
	var param0 mt.Content

	p0Map = make(param0Map)
	p0SrvMap = param0SrvMap{
		mt.Unknown: struct {
			name   string
			param0 mt.Content
		}{
			param0: mt.Unknown,
		},
		mt.Air: struct {
			name   string
			param0 mt.Content
		}{
			param0: mt.Air,
		},
		mt.Ignore: struct {
			name   string
			param0 mt.Content
		}{
			param0: mt.Ignore,
		},
	}

	for _, cc := range conns {
		<-cc.done()
		for _, def := range cc.nodeDefs {
			if p0Map[cc.name] == nil {
				p0Map[cc.name] = map[mt.Content]mt.Content{
					mt.Unknown: mt.Unknown,
					mt.Air:     mt.Air,
					mt.Ignore:  mt.Ignore,
				}
			}

			p0Map[cc.name][def.Param0] = param0
			p0SrvMap[param0] = struct {
				name   string
				param0 mt.Content
			}{
				name:   cc.name,
				param0: def.Param0,
			}

			def.Param0 = param0
			oldName := def.Name // copy string to use later
			prepend(cc.mediaPool, &def.Name)
			prepend(cc.mediaPool, &def.Mesh)
			for i := range def.Tiles {
				prependTexture(cc.mediaPool, &def.Tiles[i].Texture)
			}
			for i := range def.OverlayTiles {
				prependTexture(cc.mediaPool, &def.OverlayTiles[i].Texture)
			}
			for i := range def.SpecialTiles {
				prependTexture(cc.mediaPool, &def.SpecialTiles[i].Texture)
			}
			prependTexture(cc.mediaPool, &def.Palette)
			for k, v := range def.ConnectTo {
				def.ConnectTo[k] = p0Map[cc.name][v]
			}
			prepend(cc.mediaPool, &def.FootstepSnd.Name)
			prepend(cc.mediaPool, &def.DiggingSnd.Name)
			prepend(cc.mediaPool, &def.DugSnd.Name)
			prepend(cc.mediaPool, &def.DigPredict)
			nodeDefs = append(nodeDefs, def)

			param0++
			if param0 >= mt.Unknown && param0 <= mt.Ignore {
				param0 = mt.Ignore + 1
			}

			// add nodeid (if reqested)
			addNodeId(oldName, def.Param0)
		}
	}

	return
}

func muxMedia(conns []*contentConn) []mediaFile {
	var media []mediaFile

	for _, cc := range conns {
		<-cc.done()
		for _, f := range cc.media {
			prepend(cc.mediaPool, &f.name)
			media = append(media, f)
		}
	}

	return media
}

func muxRemotes(conns []*contentConn) []string {
	remotes := make(map[string]struct{})

	for _, cc := range conns {
		<-cc.done()
		for _, v := range cc.remotes {
			remotes[v] = struct{}{}
		}
	}

	urls := make([]string, 0, len(remotes))
	for remote := range remotes {
		urls = append(urls, remote)
	}

	return urls
}

func muxContent(userName string) (itemDefs []mt.ItemDef, aliases []struct{ Alias, Orig string }, nodeDefs []mt.NodeDef, p0Map param0Map, p0SrvMap param0SrvMap, media []mediaFile, remotes []string, err error) {
	var conns []*contentConn

PoolLoop:
	for _, pool := range PoolServers() {
		var addr *net.UDPAddr

		for name, srv := range pool {
			addr, err = net.ResolveUDPAddr("udp", srv.Addr)
			if err != nil {
				continue
			}

			var conn *net.UDPConn
			conn, err = net.DialUDP("udp", nil, addr)
			if err != nil {
				continue
			}

			var cc *contentConn
			cc, err = connectContent(conn, name, userName, srv.MediaPool)
			if err != nil {
				continue
			}
			defer cc.Close()

			conns = append(conns, cc)
			continue PoolLoop
		}

		// There's a pool with no reachable servers.
		// We can't safely let clients join.
		return
	}

	itemDefs, aliases = muxItemDefs(conns)
	nodeDefs, p0Map, p0SrvMap = muxNodeDefs(conns)
	media = muxMedia(conns)
	remotes = muxRemotes(conns)
	return
}

func (sc *ServerConn) globalParam0(p0 *mt.Content) {
	clt := sc.client()
	if clt != nil && clt.p0Map != nil {
		if clt.p0Map[sc.name] != nil {
			*p0 = clt.p0Map[sc.name][*p0]
		}
	}
}

func (cc *ClientConn) srvParam0(p0 *mt.Content) string {
	if cc.p0SrvMap != nil {
		srv := cc.p0SrvMap[*p0]
		*p0 = srv.param0
		return srv.name
	}

	return ""
}

func isDefaultNode(s string) bool {
	list := []string{
		"",
		"air",
		"unknown",
		"ignore",
	}

	for _, s2 := range list {
		if s == s2 {
			return true
		}
	}

	return false
}

func prependRaw(prep string, s *string, isTexture bool) {
	if !isDefaultNode(*s) {
		subs := disallowedChars.Split(*s, -1)
		seps := disallowedChars.FindAllString(*s, -1)

		for i, sub := range subs {
			if !isTexture || strings.Contains(sub, ".") {
				subs[i] = prep + "_" + sub
			}
		}

		*s = ""
		for i, sub := range subs {
			*s += sub
			if i < len(seps) {
				*s += seps[i]
			}
		}
	}
}

func prepend(prep string, s *string) {
	prependRaw(prep, s, false)
}

func prependTexture(prep string, t *mt.Texture) {
	s := string(*t)
	prependRaw(prep, &s, true)
	*t = mt.Texture(s)
}

func (sc *ServerConn) prependInv(inv mt.Inv) {
	for k, l := range inv {
		for i := range l.Stacks {
			prepend(sc.mediaPool, &inv[k].InvList.Stacks[i].Name)
		}
	}
}

func (sc *ServerConn) prependHUD(t mt.HUDType, cmdIface mt.ToCltCmd) {
	pa := func(cmd *mt.ToCltAddHUD) {
		switch t {
		case mt.StatbarHUD:
			prepend(sc.mediaPool, &cmd.Text2)
			fallthrough
		case mt.ImgHUD:
			fallthrough
		case mt.ImgWaypointHUD:
			fallthrough
		case mt.ImgWaypointHUD + 1:
			prepend(sc.mediaPool, &cmd.Text)
		}
	}

	pc := func(cmd *mt.ToCltChangeHUD) {
		switch t {
		case mt.StatbarHUD:
			prepend(sc.mediaPool, &cmd.Text2)
			fallthrough
		case mt.ImgHUD:
			fallthrough
		case mt.ImgWaypointHUD:
			fallthrough
		case mt.ImgWaypointHUD + 1:
			prepend(sc.mediaPool, &cmd.Text)
		}
	}

	switch cmd := cmdIface.(type) {
	case *mt.ToCltAddHUD:
		pa(cmd)
	case *mt.ToCltChangeHUD:
		pc(cmd)
	}
}
