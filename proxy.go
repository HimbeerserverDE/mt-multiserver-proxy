/*
Package proxy is a minetest reverse proxy for multiple servers.
It also provides an API for plugins.
*/
package proxy

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
)

const (
	serializeVer       = 29
	protoVer           = 42
	versionString      = "5.7.0"
	maxPlayerNameLen   = 20
	bytesPerMediaBunch = 5000
)

var playerNameChars = regexp.MustCompile("^[a-zA-Z0-9-_]+$")

var proxyDir string
var proxyDirOnce sync.Once

// Path prepends the directory the executable is in to the given path.
// It follows symlinks to the executable.
func Path(path ...string) string {
	proxyDirOnce.Do(func() {
		executable, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}

		proxyDir = filepath.Dir(executable)
	})

	return proxyDir + "/" + strings.Join(path, "")
}

// Version returns the version string of the running instance.
func Version() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	return info.Main.Version, true
}

func init() {
	version, ok := Version()
	if !ok {
		log.Fatal("unable to retrieve proxy version")
	}

	log.Println("version:", version)
}
