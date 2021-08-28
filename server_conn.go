package main

import (
	"errors"
	"fmt"
	"log"
	"net"
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
}

func (sc *serverConn) client() *clientConn { return sc.clt }

func (sc *serverConn) init() <-chan struct{} { return sc.initCh }

func (sc *serverConn) log(dir, msg string) {
	log.Printf("{←|⇶} %s {%s} %s", dir, sc.name, msg)
}

func handleSrv(sc *serverConn) {
	if sc.client() == nil {
		sc.log("-->", "no associated client")
	}

	go func() {
		for sc.state == csCreated {
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
			if sc.client() != nil {
				ack, _ := sc.client().SendCmd(cmd)
				<-ack
				sc.client().Close()
			}
		case *mt.ToCltAcceptAuth:
			sc.auth.method = 0
			sc.SendCmd(&mt.ToSrvInit2{Lang: sc.client().lang})
		case *mt.ToCltDenySudoMode:
			sc.log("<--", "deny sudo")
		case *mt.ToCltAcceptSudoMode:
			sc.log("<--", "accept sudo")
			sc.state++
		}
	}
}
