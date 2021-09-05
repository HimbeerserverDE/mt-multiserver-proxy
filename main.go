package main

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/anon55555/mt"
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	var err error
	switch conf().AuthBackend {
	case "sqlite3":
		authIface = authSQLite3{}
	default:
		log.Fatal("{←|⇶} invalid auth backend")
	}

	addr, err := net.ResolveUDPAddr("udp", conf().BindAddr)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	l := listen(pc)
	defer l.close()

	log.Print("{←|⇶} listen ", l.addr())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		<-sig

		l.mu.Lock()
		defer l.mu.Unlock()

		var wg sync.WaitGroup
		wg.Add(len(l.clts))

		for cc := range l.clts {
			go func(cc *clientConn) {
				ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.Shutdown})
				select {
				case <-cc.Closed():
				case <-ack:
					cc.Close()
				}

				<-cc.server().Closed()
				cc.mu.Lock()
				cc.srv = nil
				cc.mu.Unlock()

				wg.Done()
			}(cc)
		}

		wg.Wait()
		os.Exit(0)
	}()

	for {
		cc, err := l.accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Print("{←|⇶} stop listening")
				break
			}

			log.Print("{←|⇶} ", err)
			continue
		}

		go func() {
			<-cc.init()
			cc.log("<->", "handshake completed")

			if len(conf().Servers) == 0 {
				cc.log("<--", "no servers")
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

			addr, err := net.ResolveUDPAddr("udp", conf().Servers[0].Addr)
			if err != nil {
				cc.log("<--", "address resolution fail")
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
				cc.log("<--", "connection fail")

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

			connect(conn, conf().Servers[0].Name, cc)
		}()
	}

	select {}
}
