package proxy

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const latestSerializeVer = 28
const latestProtoVer = 39
const maxPlayerNameLen = 20
const playerNameChars = "^[a-zA-Z0-9-_]+$"
const bytesPerMediaBunch = 5000

const defaultCmdPrefix = ">"
const defaultSendInterval = 0.09
const defaultUserLimit = 10
const defaultAuthBackend = "sqlite3"
const defaultBindAddr = ":40000"

var config Config
var configMu sync.RWMutex

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
	MapRange uint32
}

func Conf() Config {
	configMu.RLock()
	defer configMu.RUnlock()

	return config
}

func LoadConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	oldConf := config

	config.CmdPrefix = defaultCmdPrefix
	config.SendInterval = defaultSendInterval
	config.UserLimit = defaultUserLimit
	config.AuthBackend = defaultAuthBackend
	config.BindAddr = defaultBindAddr

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
