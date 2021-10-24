package proxy

import (
	"errors"
	"net"
	"time"
)

var authIface authBackend
var ErrAuthBackendExists = errors.New("auth backend already set")

type user struct {
	name      string
	salt      []byte
	verifier  []byte
	timestamp time.Time
}

type ban struct {
	addr string
	name string
}

type authBackend interface {
	Exists(name string) bool
	Passwd(name string) (salt, verifier []byte, err error)
	SetPasswd(name string, salt, verifier []byte) error
	LastSrv(name string) (string, error)
	SetLastSrv(name, srv string) error
	Timestamp(name string) (time.Time, error)
	Import(in []user)
	Export() ([]user, error)

	Ban(addr, name string) error
	Unban(id string) error
	Banned(addr *net.UDPAddr) bool
	ImportBans(in []ban)
	ExportBans() ([]ban, error)
}

func setAuthBackend(ab authBackend) error {
	if authIface != nil {
		return ErrAuthBackendExists
	}

	authIface = ab
	return nil
}
