package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/HimbeerserverDE/srp"
	"github.com/anon55555/mt"
	"github.com/anon55555/mt/rudp"
)

type mediaFile struct {
	name       string
	base64SHA1 string
	data       []byte
}

type contentConn struct {
	mt.Peer

	state          clientState
	name, userName string
	doneCh         chan struct{}

	auth struct {
		method              mt.AuthMethods
		salt, srpA, a, srpK []byte
	}

	itemDefs []mt.ItemDef
	aliases  []struct{ Alias, Orig string }

	nodeDefs []mt.NodeDef

	media []mediaFile
}

func (cc *contentConn) done() <-chan struct{} { return cc.doneCh }

func (cc *contentConn) log(dir, msg string) {
	log.Printf("{←|⇶} %s {%s} %s", dir, cc.name, msg)
}

func handleContent(cc *contentConn) {
	defer close(cc.doneCh)

	go func() {
		for cc.state == csCreated {
			cc.SendCmd(&mt.ToSrvInit{
				SerializeVer: latestSerializeVer,
				MinProtoVer:  latestProtoVer,
				MaxProtoVer:  latestProtoVer,
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
				break
			}

			cc.log("-->", err.Error())
			continue
		}

		switch cmd := pkt.Cmd.(type) {
		case *mt.ToCltHello:
			if cc.auth.method != 0 {
				cc.log("<--", "unexpected authentication")
				cc.Close()
				break
			}

			cc.state++

			if cmd.AuthMethods&mt.FirstSRP != 0 {
				cc.auth.method = mt.FirstSRP
			} else {
				cc.auth.method = mt.SRP
			}

			if cmd.SerializeVer != latestSerializeVer {
				cc.log("<--", "invalid serializeVer")
				break
			}

			switch cc.auth.method {
			case mt.SRP:
				cc.auth.srpA, cc.auth.a, err = srp.InitiateHandshake()
				if err != nil {
					cc.log("-->", err.Error())
					break
				}

				cc.SendCmd(&mt.ToSrvSRPBytesA{
					A:      cc.auth.srpA,
					NoSHA1: true,
				})
			case mt.FirstSRP:
				salt, verifier, err := srp.NewClient([]byte(cc.userName), []byte{})
				if err != nil {
					cc.log("-->", err.Error())
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
				cc.log("<--", "multiple authentication attempts")
				break
			}

			cc.auth.srpK, err = srp.CompleteHandshake(cc.auth.srpA, cc.auth.a, []byte(cc.userName), []byte{}, cmd.Salt, cmd.B)
			if err != nil {
				cc.log("-->", err.Error())
				break
			}

			M := srp.ClientProof([]byte(cc.userName), cmd.Salt, cc.auth.srpA, cmd.B, cc.auth.srpK)
			if M == nil {
				cc.log("<--", "SRP safety check fail")
				break
			}

			cc.SendCmd(&mt.ToSrvSRPBytesM{
				M: M,
			})
		case *mt.ToCltDisco:
			cc.log("<--", fmt.Sprintf("deny access %+v", cmd))
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
				cc.media = append(cc.media, mediaFile{
					name:       f.Name,
					base64SHA1: f.Base64SHA1,
				})

				filenames = append(filenames, f.Name)
			}

			cc.SendCmd(&mt.ToSrvReqMedia{Filenames: filenames})
		case *mt.ToCltMedia:
			for _, f := range cmd.Files {
				for _, af := range cc.media {
					if af.name == f.Name {
						af.data = f.Data
						break
					}
				}
			}

			if cmd.I == cmd.N-1 {
				cc.Close()
			}
		}
	}
}

func (cc *clientConn) sendMedia(filenames []string) {
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
			cc.log("-->", "request unknown media file")
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
	var wg sync.WaitGroup

	itemDefs = append(itemDefs, mt.ItemDef{
		Type:       mt.ToolItem,
		InvImg:     "blank.png",
		WieldScale: [3]float32{1, 1, 1},
		StackMax:   1,
		Usable:     true,
		ToolCaps: mt.ToolCaps{
			NonNil: true,
		},
		PointRange: 4,
	})

	for _, cc := range conns {
		wg.Add(1)

		prepend := func(s *string) {
			if *s != "" {
				*s = cc.name + "_" + *s
			}
		}
		prependTexture := func(s *mt.Texture) {
			if *s != "" {
				*s = mt.Texture(cc.name) + "_" + *s
			}
		}

		go func() {
			<-cc.done()
			for _, def := range cc.itemDefs {
				if def.Name == "" {
					def.Name = "hand"
				}
				prepend(&def.Name)

				prependTexture(&def.InvImg)
				prependTexture(&def.WieldImg)
				prepend(&def.PlacePredict)
				prepend(&def.PlaceSnd.Name)
				prepend(&def.PlaceFailSnd.Name)
				prependTexture(&def.Palette)
				prependTexture(&def.InvOverlay)
				prependTexture(&def.WieldOverlay)
				itemDefs = append(itemDefs, def)
			}

			for _, alias := range cc.aliases {
				aliases = append(aliases, struct{ Alias, Orig string }{
					Alias: cc.name + "_" + alias.Alias,
					Orig:  cc.name + "_" + alias.Orig,
				})
			}

			wg.Done()
		}()
	}

	wg.Wait()
	return itemDefs, aliases
}

func muxNodeDefs(conns []*contentConn) (nodeDefs []mt.NodeDef, p0Map param0Map, p0SrvMap param0SrvMap) {
	var wg sync.WaitGroup
	var param0 mt.Content

	p0Map = make(param0Map)
	p0SrvMap = make(param0SrvMap)

	for _, cc := range conns {
		wg.Add(1)

		prepend := func(s *string) {
			if *s != "" {
				*s = cc.name + "_" + *s
			}
		}
		prependTexture := func(s *mt.Texture) {
			if *s != "" {
				*s = mt.Texture(cc.name) + "_" + *s
			}
		}

		go func() {
			<-cc.done()
			for _, def := range cc.nodeDefs {
				if p0Map[cc.name] == nil {
					p0Map[cc.name] = make(map[mt.Content]mt.Content)
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
				prepend(&def.Name)
				prepend(&def.Mesh)
				for i := range def.Tiles {
					prependTexture(&def.Tiles[i].Texture)
				}
				for i := range def.OverlayTiles {
					prependTexture(&def.OverlayTiles[i].Texture)
				}
				for i := range def.SpecialTiles {
					prependTexture(&def.SpecialTiles[i].Texture)
				}
				prependTexture(&def.Palette)
				for k, v := range def.ConnectTo {
					def.ConnectTo[k] = p0Map[cc.name][v]
				}
				prepend(&def.FootstepSnd.Name)
				prepend(&def.DiggingSnd.Name)
				prepend(&def.DugSnd.Name)
				prepend(&def.DigPredict)
				nodeDefs = append(nodeDefs, def)

				param0++
				if param0 >= mt.Unknown || param0 <= mt.Ignore {
					param0 = mt.Ignore + 1
				}
			}

			wg.Done()
		}()
	}

	wg.Wait()
	return
}

func muxMedia(conns []*contentConn) []mediaFile {
	var media []mediaFile
	var wg sync.WaitGroup

	for _, cc := range conns {
		wg.Add(1)

		prepend := func(s *string) {
			if *s != "" {
				*s = cc.name + "_" + *s
			}
		}

		go func() {
			<-cc.done()
			for _, f := range cc.media {
				prepend(&f.name)
				media = append(media, f)
			}

			wg.Done()
		}()
	}

	wg.Wait()
	return media
}

func muxContent(userName string) (itemDefs []mt.ItemDef, aliases []struct{ Alias, Orig string }, nodeDefs []mt.NodeDef, p0Map param0Map, p0SrvMap param0SrvMap, media []mediaFile, err error) {
	var conns []*contentConn
	for _, srv := range conf.Servers {
		var addr *net.UDPAddr
		addr, err = net.ResolveUDPAddr("udp", srv.Addr)
		if err != nil {
			return
		}

		var conn *net.UDPConn
		conn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			return
		}

		cc := connectContent(conn, srv.Name, userName)
		defer cc.Close()

		conns = append(conns, cc)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		itemDefs, aliases = muxItemDefs(conns)
	}()

	go func() {
		defer wg.Done()
		nodeDefs, p0Map, p0SrvMap = muxNodeDefs(conns)
	}()

	go func() {
		defer wg.Done()
		media = muxMedia(conns)
	}()

	wg.Wait()
	return
}
