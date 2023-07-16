package proxy

import (
	"errors"
	"net"
	"strings"
	"time"
)

var authIface AuthBackend

var (
	ErrAuthBackendExists   = errors.New("auth backend already set")
	ErrBanNotSupported     = errors.New("auth backend does not support bans")
	ErrInvalidSRPHeader    = errors.New("encoded password is not SRP")
	ErrLastSrvNotSupported = errors.New("auth backend does not support server information")
)

type User struct {
	Name      string
	Salt      []byte
	Verifier  []byte
	Timestamp time.Time
}

type Ban struct {
	Addr string
	Name string
}

type AuthBackend interface {
	Exists(name string) bool
	Passwd(name string) (salt, verifier []byte, err error)
	SetPasswd(name string, salt, verifier []byte) error
	LastSrv(name string) (string, error)
	SetLastSrv(name, srv string) error
	Timestamp(name string) (time.Time, error)
	Import(in []User) error
	Export() ([]User, error)

	Ban(addr, name string) error
	Unban(id string) error
	Banned(addr *net.UDPAddr) bool
	ImportBans(in []Ban) error
	ExportBans() ([]Ban, error)
}

func setAuthBackend(ab AuthBackend) error {
	if authIface != nil {
		return ErrAuthBackendExists
	}

	authIface = ab
	return nil
}

func encodeVerifierAndSalt(salt, verifier []byte) string {
	return "#1#" + b64.EncodeToString(salt) + "#" + b64.EncodeToString(verifier)
}

func decodeVerifierAndSalt(encodedPasswd string) ([]byte, []byte, error) {
	if !strings.HasPrefix(encodedPasswd, "#1#") {
		return nil, nil, ErrInvalidSRPHeader
	}

	salt, err := b64.DecodeString(strings.Split(encodedPasswd, "#")[2])
	if err != nil {
		return nil, nil, err
	}

	verifier, err := b64.DecodeString(strings.Split(encodedPasswd, "#")[3])
	if err != nil {
		return nil, nil, err
	}

	return salt, verifier, nil
}
