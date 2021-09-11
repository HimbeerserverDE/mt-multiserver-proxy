package proxy

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const defaultCmdPrefix = ">"
const defaultSendInterval = 0.09
const defaultUserLimit = 10
const defaultAuthBackend = "sqlite3"
const defaultBindAddr = ":40000"
const defaultListInterval = 300

var config Config
var configMu sync.RWMutex

// A Config contains information from the configuration file
// that affects the way the proxy works.
type Config struct {
	NoPlugins     bool
	CmdPrefix     string
	RequirePasswd bool
	SendInterval  float32
	UserLimit     int
	AuthBackend   string
	BindAddr      string
	Servers       []struct {
		Name string
		Addr string
	}
	CSMRF struct {
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
	configMu.RLock()
	defer configMu.RUnlock()

	return config
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
	config.BindAddr = defaultBindAddr
	config.Groups = make(map[string][]string)
	config.UserGroups = make(map[string]string)
	config.List.Interval = defaultListInterval

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	path := filepath.Dir(executable) + "/config.json"
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
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

	log.Print("{←|⇶} load config")
	return nil
}
