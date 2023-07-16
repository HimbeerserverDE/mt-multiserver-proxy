/*
mt-auth-convert converts between authentication backends.

Usage:

	mt-auth-convert from to

where from is the format to convert from
and to is the format to convert to
*/
package main

import (
	"log"
	"os"

	proxy "github.com/HimbeerserverDE/mt-multiserver-proxy"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: mt-auth-convert from to")
	}

	var inBackend proxy.AuthBackend
	switch os.Args[1] {
	case "files":
		inBackend = proxy.AuthFiles{}
	case "mtsqlite3":
		var err error
		inBackend, err = proxy.NewAuthMTSQLite3()
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("invalid input auth backend")
	}

	var outBackend proxy.AuthBackend
	switch os.Args[2] {
	case "files":
		outBackend = proxy.AuthFiles{}
	case "mtsqlite3":
		var err error
		outBackend, err = proxy.NewAuthMTSQLite3()
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("invalid output auth backend")
	}

	if err := convert(outBackend, inBackend); err != nil {
		log.Fatal(err)
	}

	log.Print("conversion successful")
}

func convert(dst, src proxy.AuthBackend) error {
	users, err := src.Export()
	if err != nil {
		return err
	}

	if err := dst.Import(users); err != nil {
		return err
	}

	return nil
}
