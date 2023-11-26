package proxy

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"plugin"
	"sync"
)

var pluginsOnce sync.Once

func BuildPlugin() error {
	version, ok := Version()
	if !ok {
		return fmt.Errorf("unable to retrieve proxy version")
	}

	pathVer := "github.com/HimbeerserverDE/mt-multiserver-proxy@" + version

	if err := goCmd("get", "-u", pathVer); err != nil {
		return err
	}

	if err := goCmd("mod", "tidy"); err != nil {
		return err
	}

	if err := goCmd("build", "-buildmode=plugin"); err != nil {
		return err
	}

	return nil
}

func loadPlugins() {
	pluginsOnce.Do(openPlugins)
}

func openPlugins() {
	path := Path("plugins")
	os.Mkdir(path, 0777)

	dir, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, pl := range dir {
		if pl.IsDir() && !Conf().NoAutoPlugins {
			plPath := path + "/" + pl.Name()

			wd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}

			if err := os.Chdir(plPath); err != nil {
				log.Fatal(err)
			}

			if err := BuildPlugin(); err != nil {
				log.Fatal(err)
			}

			if err := os.Chdir(wd); err != nil {
				log.Fatal(err)
			}

			_, err = plugin.Open(path + "/" + pl.Name() + "/" + pl.Name() + ".so")
			if err != nil {
				log.Print(err)
				continue
			}
		} else if !pl.IsDir() {
			_, err := plugin.Open(path + "/" + pl.Name())
			if err != nil {
				log.Print(err)
				continue
			}
		}
	}

	log.Print("load plugins")
}

func goCmd(args ...string) error {
	cmd := exec.Command("go", args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
