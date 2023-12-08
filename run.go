package proxy

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var runOnce sync.Once

// Run initializes the proxy and starts the main listener loop.
// It blocks forever.
func Run() {
	runOnce.Do(runFunc)
}

func runFunc() {
	if !Conf().NoPlugins {
		loadPlugins()
	}

	var err error
	switch Conf().AuthBackend {
	case "files":
		setAuthBackend(AuthFiles{})
	case "mtsqlite3":
		ab, err := NewAuthMTSQLite3()
		if err != nil {
			log.Fatal(err)
		}

		setAuthBackend(ab)
	case "mtpostgresql":
		ab, err := NewAuthMTPostgreSQL(Conf().AuthPostgresConn)
		if err != nil {
			log.Fatal(err)
		}

		setAuthBackend(ab)
	default:
		log.Fatal("invalid auth backend")
	}

	addr, err := net.ResolveUDPAddr("udp", Conf().BindAddr)
	if err != nil {
		log.Fatal(err)
	}

	pc, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	l := listen(pc)
	defer l.Close()

	log.Println("listen", l.Addr())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		<-sig

		if Conf().List.Enable {
			if err := announce(listRm); err != nil {
				log.Print(err)
			}
		}

		clts := Clts()

		var wg sync.WaitGroup
		wg.Add(len(clts))

		for cc := range clts {
			go func(cc *ClientConn) {
				sc := cc.server()

				cc.Kick("Proxy shutting down.")
				<-cc.Closed()

				if sc != nil {
					<-sc.Closed()
				}

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
				log.Print("stop listening")
				break
			}

			log.Print(err)
			continue
		}

		go func() {
			<-cc.Init()
			cc.Log("<->", "handshake completed")

			conf := Conf()
			if len(conf.Servers) == 0 {
				cc.Log("<-", "no servers")
				cc.Kick("No servers are configured.")
				return
			}

			srvName, srv := conf.DefaultServerInfo()
			lastSrv, err := authIface.LastSrv(cc.Name())
			if err == nil && !conf.ForceDefaultSrv && lastSrv != srvName {
				choice, ok := conf.RandomGroupServer(lastSrv)
				if !ok {
					cc.Log("<-", "inexistent previous server")
				}

				srvName = choice
				srv, _ = conf.Servers[choice] // Existence already checked.
			}

			addr, err := net.ResolveUDPAddr("udp", srv.Addr)
			if err != nil {
				cc.Log("<-", "address resolution fail")
				cc.Kick("Server address resolution failed.")
				return
			}

			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				cc.Log("<-", "connection fail")
				cc.Kick("Server connection failed.")
				return
			}

			connect(conn, srvName, cc)
		}()
	}

	select {}
}
