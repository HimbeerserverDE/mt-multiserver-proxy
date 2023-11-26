/*
mt-build-plugin builds a plugin using the proxy version the tool was built for.

Usage:

	mt-build-plugin
*/
package main

import (
	"log"
	"os"
	"os/exec"

	proxy "github.com/HimbeerserverDE/mt-multiserver-proxy"
)

func main() {
	version, ok := proxy.Version()
	if !ok {
		log.Fatal("unable to retrieve proxy version")
	}

	if err := proxy.BuildPlugin(); err != nil {
		log.Fatal(err)
	}
}

func goCmd(args ...string) error {
	cmd := exec.Command("go", args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
