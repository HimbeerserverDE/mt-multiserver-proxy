/*
mt-build-plugin builds a plugin using the proxy version the tool was built for.

Usage:

	mt-build-plugin
*/
package main

import (
	"log"
	"os/exec"

	proxy "github.com/HimbeerserverDE/mt-multiserver-proxy"
)

func main() {
	version, ok := proxy.Version()
	if !ok {
		log.Fatal("unable to retrieve proxy version")
	}

	log.Println("version:", version)

	pathVer := "github.com/HimbeerserverDE/mt-multiserver-proxy@" + version

	if err := exec.Command("go", "get", "-u", pathVer); err != nil {
		log.Fatalln("error updating proxy dependency:", err)
	}

	if err := exec.Command("go", "build", "-buildmode=plugin"); err != nil {
		log.Fatalln("error building plugin:", err)
	}
}
