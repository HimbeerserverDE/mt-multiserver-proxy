package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

const (
	defaultCmdPrefix    = ">"
	defaultSendInterval = 0.09
	defaultUserLimit    = 10
	defaultAuthBackend  = "files"
	defaultTelnetAddr   = "[::1]:40010"
	defaultBindAddr     = ":40000"
	defaultListInterval = 300
)

var config Config
var configMu sync.RWMutex

var loadConfigOnce sync.Once

type Server struct {
	Addr      string
	MediaPool string
	Fallbacks []string

	dynamic bool
}

// A Config contains information from the configuration file
// that affects the way the proxy works.
type Config struct {
	NoPlugins       bool
	CmdPrefix       string
	RequirePasswd   bool
	SendInterval    float32
	UserLimit       int
	AuthBackend     string
	NoTelnet        bool
	TelnetAddr      string
	BindAddr        string
	Servers         map[string]Server
	ForceDefaultSrv bool
	FallbackServers []string
	CSMRF           struct {
		NoCSMs          bool
		ChatMsgs        bool
		ItemDefs        bool
		NodeDefs        bool
		NoLimitMapRange bool
		PlayerList      bool
	}
	MapRange   uint32
	DropCSMRF  bool
	Groups     map[string][]string
	UserGroups map[string]string
	List       struct {
		Enable   bool
		Addr     string
		Interval int

		Name     string
		Desc     string
		URL      string
		Creative bool
		Dmg      bool
		PvP      bool
		Game     string
		FarNames bool
		Mods     []string
	}
}

// Conf returns a copy of the Config used by the proxy.
// Any modifications will not affect the original Config.
func Conf() Config {
	loadConfigOnce.Do(func() {
		if err := LoadConfig(); err != nil {
			log.Fatal(err)
		}
	})

	configMu.RLock()
	defer configMu.RUnlock()

	return config
}

// PoolServers returns all media pools and their member servers.
func PoolServers() map[string]map[string]Server {
	pools := make(map[string]map[string]Server)
	for name, srv := range Conf().Servers {
		if pools[srv.MediaPool] == nil {
			pools[srv.MediaPool] = make(map[string]Server)
		}

		pools[srv.MediaPool][name] = srv
	}

	return pools
}

// AddServer dynamically configures a new Server at runtime.
// Servers added in this way are ephemeral and will be lost
// when the proxy shuts down.
// The server must be part of a media pool with at least one
// other member. At least one of the other members always
// needs to be reachable.
// WARNING: Reloading the config will not overwrite servers
// added using this function. The server definition from the
// configuration file will silently be ignored.
func AddServer(name string, s Server) bool {
	configMu.Lock()
	defer configMu.Unlock()

	s.dynamic = true

	if _, ok := config.Servers[name]; ok {
		return false
	}

	var poolMembers bool
	for _, srv := range config.Servers {
		if srv.MediaPool == s.MediaPool {
			poolMembers = true
		}
	}

	if !poolMembers {
		return false
	}

	config.Servers[name] = s
	return true
}

// RmServer deletes a Server from the Config at runtime.
// Only servers added using AddServer can be deleted at runtime.
// Returns true on success or if the server doesn't exist.
func RmServer(name string) bool {
	configMu.Lock()
	defer configMu.Unlock()

	s, ok := config.Servers[name]
	if !ok {
		return true
	}

	if !s.dynamic {
		return false
	}

	// Can't remove server if players are connected to it
	for cc := range Clts() {
		if cc.ServerName() == name {
			return false
		}
	}

	delete(config.Servers, name)
	return true
}

// DefaultServerInfo returns both the name of the default server
// and information about it. The return values are uninitialized
// if no servers exist.
func (cnf Config) DefaultServerInfo() (string, Server) {
	for name, srv := range Conf().Servers {
		return name, srv
	}

	// No servers are configured.
	return "", Server{}
}

// DefaultServerName returns the name of the default server.
// If no servers exist it returns an empty string.
func (cnf Config) DefaultServerName() string {
	name, _ := cnf.DefaultServerInfo()
	return name
}

// DefaultServer returns information about the default server.
// If no servers exist the returned struct will be uninitialized.
// This is a faster shortcut for Config.Servers[Config.DefaultServerName()].
// You should thus only use this method or the DefaultServerInfo method.
func (cnf Config) DefaultServer() Server {
	_, srv := cnf.DefaultServerInfo()
	return srv
}

// FallbackServers returns a slice of server names that
// a server can fall back to.
func FallbackServers(server string) []string {
	conf := Conf()

	srv, ok := conf.Servers[server]
	if !ok {
		return nil
	}

	fallbacks := srv.Fallbacks

	// global fallbacks
	if len(conf.FallbackServers) == 0 {
		if len(conf.Servers) == 0 {
			return fallbacks
		}

		return append(fallbacks, conf.DefaultServerName())
	} else {
		return append(fallbacks, conf.FallbackServers...)
	}
}

// LoadConfig attempts to parse the configuration file.
// It leaves the config unchanged if there is an error
// and returns the error.
func LoadConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	oldConf := config

	config.CmdPrefix = defaultCmdPrefix
	config.SendInterval = defaultSendInterval
	config.UserLimit = defaultUserLimit
	config.AuthBackend = defaultAuthBackend
	config.TelnetAddr = defaultTelnetAddr
	config.BindAddr = defaultBindAddr
	config.FallbackServers = make([]string, 0)
	config.Groups = make(map[string][]string)
	config.UserGroups = make(map[string]string)
	config.List.Interval = defaultListInterval

	f, err := os.OpenFile(Path("config.json"), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		config = oldConf
		return err
	}
	defer f.Close()

	if fi, _ := f.Stat(); fi.Size() == 0 {
		f.WriteString("{\n\t\n}\n")
		f.Seek(0, os.SEEK_SET)
	}

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&config); err != nil {
		config = oldConf
		return err
	}

	// Dynamic servers shouldn't be deleted silently.
	for name, srv := range oldConf.Servers {
		if srv.dynamic {
			if _, ok := config.Servers[name]; ok {
				config = oldConf
				return fmt.Errorf("duplicate server %s", name)
			}

			config.Servers[name] = srv
		} else {
			if _, ok := config.Servers[name]; ok {
				continue
			}

			for cc := range Clts() {
				if cc.ServerName() == name {
					config = oldConf
					return fmt.Errorf("can't delete server %s with players", name)
				}
			}
		}
	}

	for name, srv := range config.Servers {
		if srv.MediaPool == "" {
			s := config.Servers[name]
			s.MediaPool = name
			config.Servers[name] = s
		}
	}

	log.Print("load config")
	return nil
}
