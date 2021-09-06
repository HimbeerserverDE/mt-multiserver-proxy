package main

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/HimbeerserverDE/mt-multiserver-proxy"
	"github.com/anon55555/mt"
)

func main() {
	if err := proxy.LoadConfig(); err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	if !proxy.Conf().NoPlugins {
		proxy.LoadPlugins()
	}

	var err error
	switch proxy.Conf().AuthBackend {
	case "sqlite3":
		proxy.SetAuthBackend(proxy.AuthSQLite3{})
	default:
		log.Fatal("{←|⇶} invalid auth backend")
	}

	addr, err := net.ResolveUDPAddr("udp", proxy.Conf().BindAddr)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	l := proxy.Listen(pc)
	defer l.Close()

	log.Print("{←|⇶} listen ", l.Addr())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		<-sig

		clts := l.Clts()

		var wg sync.WaitGroup
		wg.Add(len(clts))

		for cc := range clts {
			go func(cc *proxy.ClientConn) {
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.Shutdown})
				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				wg.Done()
			}(cc)
		}

		wg.Wait()
		os.Exit(0)
	}()

	for {
		cc, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Print("{←|⇶} stop listening")
				break
			}

			log.Print("{←|⇶} ", err)
			continue
		}

		go func() {
			<-cc.Init()
			cc.Log("<->", "handshake completed")

			if len(proxy.Conf().Servers) == 0 {
				cc.Log("<--", "no servers")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{
					Reason: mt.Custom,
					Custom: "No servers are configured.",
				})
				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			addr, err := net.ResolveUDPAddr("udp", proxy.Conf().Servers[0].Addr)
			if err != nil {
				cc.Log("<--", "address resolution fail")
				ack, _ := cc.SendCmd(&mt.ToCltDisco{
					Reason: mt.Custom,
					Custom: "Server address resolution failed.",
				})
				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				cc.Log("<--", "connection fail")

				ack, _ := cc.SendCmd(&mt.ToCltDisco{
					Reason: mt.Custom,
					Custom: "Server connection failed.",
				})

				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				return
			}

			proxy.Connect(conn, proxy.Conf().Servers[0].Name, cc)
		}()
	}

	select {}
}
