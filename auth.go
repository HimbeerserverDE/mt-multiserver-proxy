package proxy

import (
	"errors"
	"time"
)

var authIface AuthBackend
var ErrAuthBackendExists = errors.New("auth backend already set")

type user struct {
	name      string
	salt      []byte
	verifier  []byte
	timestamp time.Time
}

type AuthBackend interface {
	Exists(name string) bool
	Passwd(name string) (salt, verifier []byte, err error)
	SetPasswd(name string, salt, verifier []byte) error
	Timestamp(name string) (time.Time, error)
	Import(data []user)
	Export() ([]user, error)
}

func SetAuthBackend(ab AuthBackend) error {
	if authIface != nil {
		return ErrAuthBackendExists
	}

	authIface = ab
	return nil
}
