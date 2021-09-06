package proxy

import (
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

var plugins map[*plugin.Plugin]struct{}
var pluginsOnce sync.Once

func LoadPlugins() {
	pluginsOnce.Do(loadPlugins)
}

func loadPlugins() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	path := filepath.Dir(executable) + "/plugins"
	os.Mkdir(path, 0777)

	dir, err := os.ReadDir(path)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	plugins = make(map[*plugin.Plugin]struct{})
	for _, file := range dir {
		p, err := plugin.Open(path + "/" + file.Name())
		if err != nil {
			log.Print("{←|⇶} ", err)
			continue
		}

		plugins[p] = struct{}{}
	}

	log.Print("{←|⇶} load plugins")
}
