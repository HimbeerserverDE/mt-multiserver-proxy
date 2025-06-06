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
		LoadPlugins()
	}

	authBackendName := Conf().AuthBackend
	switch authBackendName {
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
	case "":
		log.Fatal("invalid auth backend")
	default:
		if ab, ok := authBackends[authBackendName]; ok {
			setAuthBackend(ab)
		} else {
			log.Fatal("invalid auth backend")
		}
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

			srvName, srv := selectSrv(cc)
			if srvName == "" {
				if _, ok := conf.Servers[""]; !ok && conf.DefaultServerName() == "" {
					cc.Log("<-", "no default server")
					cc.Kick("No valid default server is configured.")
					return
				}

				srvName, srv = conf.DefaultServerInfo()
				lastSrv, err := DefaultAuth().LastSrv(cc.Name())
				if err == nil && !conf.ForceDefaultSrv && lastSrv != srvName {
					choice, ok := conf.RandomGroupServer(lastSrv)
					if !ok {
						cc.Log("<-", "inexistent previous server")
					}

					srvName = choice
					srv, _ = conf.Servers[choice] // Existence already checked.
				}
			}

			doConnect := func(srvName string, srv Server) error {
				addr, err := net.ResolveUDPAddr("udp", srv.Addr)
				if err != nil {
					cc.Log("<-", "address resolution fail")
					// cc.Kick("Server address resolution failed.")
					return err
				}

				conn, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					cc.Log("<-", "connection fail")
					// cc.Kick("Server connection failed.")
					return err
				}

				connect(conn, srvName, cc)
				return nil
			}

			if err := doConnect(srvName, srv); err != nil {
				cc.Log("<-", "connect", srvName+":", err)
				cc.SendChatMsg("Could not connect, trying fallback server. Error:", err)

				fbName := srv.Fallback
				for fbName != "" {
					var ok bool
					srv, ok = conf.Servers[fbName]
					if !ok {
						cc.Log("<-", "invalid fallback")
						continue
					}

					if err = doConnect(fbName, srv); err != nil {
						srvName = fbName
						fbName = srv.Fallback

						cc.Log("<-", "connect", srvName+":", err)
						cc.SendChatMsg("Could not connect, trying next fallback server. Error:", err.Error())
					} else {
						return
					}
				}

				cc.Kick("All upstream connections failed. Please try again later or contact the server administrator. Error: " + err.Error())
			}
		}()
	}

	select {}
}
