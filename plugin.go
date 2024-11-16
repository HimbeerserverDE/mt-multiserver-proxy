package proxy

import (
	"log"
	"os"
	"os/exec"
	"plugin"
	"sync"
)

var pluginsOnce sync.Once

func BuildPlugin() error {
	version, err := Version()
	if err != nil {
		return err
	}

	if version == "(devel)" {
		return buildPluginDev(version)
	}

	return buildPlugin(version)
}

func buildPlugin(version string) error {
	pathVer := "github.com/HimbeerserverDE/mt-multiserver-proxy@" + version

	if err := goCmd("get", pathVer); err != nil {
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

func buildPluginDev(version string) error {
	replace := "-replace=github.com/HimbeerserverDE/mt-multiserver-proxy=" + Path()
	const dropReplace = "-dropreplace=github.com/HimbeerserverDE/mt-multiserver-proxy"

	if err := goCmd("mod", "edit", replace); err != nil {
		return err
	}

	if err := goCmd("mod", "tidy"); err != nil {
		return err
	}

	if err := goCmd("build", "-buildmode=plugin"); err != nil {
		return err
	}

	if err := goCmd("mod", "edit", dropReplace); err != nil {
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

			log.Println("load auto plugin", pl.Name())
		} else if !pl.IsDir() {
			_, err := plugin.Open(path + "/" + pl.Name())
			if err != nil {
				log.Print(err)
				continue
			}

			log.Println("load comp plugin", pl.Name())
		}
	}
}

func goCmd(args ...string) error {
	cmd := exec.Command("go", args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
