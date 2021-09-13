package proxy

import (
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

var pluginsOnce sync.Once

func loadPlugins() {
	pluginsOnce.Do(openPlugins)
}

func openPlugins() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	path := filepath.Dir(executable) + "/plugins"
	os.Mkdir(path, 0777)

	dir, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range dir {
		_, err := plugin.Open(path + "/" + file.Name())
		if err != nil {
			log.Print(err)
			continue
		}
	}

	log.Print("load plugins")
}
