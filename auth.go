package main

import "time"

var authIface authBackend

type user struct {
	name      string
	salt      []byte
	verifier  []byte
	timestamp time.Time
}

type authBackend interface {
	Exists(name string) bool
	Passwd(name string) (salt, verifier []byte, err error)
	SetPasswd(name string, salt, verifier []byte) error
	Timestamp(name string) (time.Time, error)
	Import(data []user)
	Export() ([]user, error)
}
