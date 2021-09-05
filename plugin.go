package main

import (
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

var plugins []*plugin.Plugin
var pluginsMu sync.RWMutex

func loadPlugins() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	path := filepath.Dir(executable) + "/plugins"
	os.Mkdir(path, 0777)

	dir, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	pluginsMu.Lock()
	defer pluginsMu.Unlock()

	plugins = []*plugin.Plugin{}

	for _, file := range dir {
		p, err := plugin.Open(path + "/" + file.Name())
		if err != nil {
			log.Print("{←|⇶} ", err)
			continue
		}

		plugins = append(plugins, p)
	}

	log.Print("{←|⇶} load plugins")
	return nil
}
