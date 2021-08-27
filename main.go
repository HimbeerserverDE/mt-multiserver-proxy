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
	switch conf.AuthBackend {
	case "sqlite3":
		authIface = authSQLite3{}
	default:
		log.Fatal("{←|⇶} invalid auth backend")
	}

	addr, err := net.ResolveUDPAddr("udp", conf.BindAddr)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	l := listen(pc)
	defer l.close()

	log.Print("{←|⇶} listening on ", l.addr())

	clts := make(map[*clientConn]struct{})
	var mu sync.Mutex

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		<-sig

		mu.Lock()
		defer mu.Unlock()

		for cc := range clts {
			ack, _ := cc.SendCmd(&mt.ToCltDisco{Reason: mt.Shutdown})
			<-ack
			cc.Close()
		}

		os.Exit(0)
	}()

	for {
		cc, err := l.accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			log.Print("{←|⇶} ", err)
			continue
		}

		mu.Lock()
		clts[cc] = struct{}{}
		mu.Unlock()

		go func() {
			<-cc.Closed()

			mu.Lock()
			defer mu.Unlock()

			delete(clts, cc)
		}()

		go func() {
			<-cc.init()
			cc.log("<->", "handshake completed")
			// ToDo: establish serverConn
			// and start handler goroutines
		}()
	}
}
