package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

const latestSerializeVer = 28
const latestProtoVer = 39
const maxPlayerNameLen = 20
const playerNameChars = "^[a-zA-Z0-9-_]+$"
const bytesPerMediaBunch = 5000

const defaultSendInterval = 0.09
const defaultUserLimit = 10
const defaultAuthBackend = "sqlite3"
const defaultBindAddr = ":40000"

var conf Config

type Config struct {
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

func loadConfig() error {
	oldConf := conf

	conf.SendInterval = defaultSendInterval
	conf.UserLimit = defaultUserLimit
	conf.AuthBackend = defaultAuthBackend
	conf.BindAddr = defaultBindAddr

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	path := filepath.Dir(executable) + "/config.json"
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		conf = oldConf
		return err
	}
	defer f.Close()

	if fi, _ := f.Stat(); fi.Size() == 0 {
		f.WriteString("{\n\t\n}\n")
		f.Seek(0, os.SEEK_SET)
	}

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&conf); err != nil {
		conf = oldConf
		return err
	}

	log.Print("{←|⇶} load config")
	return nil
}
