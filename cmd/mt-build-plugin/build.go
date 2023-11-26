/*
mt-build-plugin builds a plugin using the proxy version the tool was built for.

Usage:

	mt-build-plugin
*/
package main

import (
	"log"

	proxy "github.com/HimbeerserverDE/mt-multiserver-proxy"
)

func main() {
	if err := proxy.BuildPlugin(); err != nil {
		log.Fatal(err)
	}
}
