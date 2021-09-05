package main

import (
	"log"
	"os"
	"path/filepath"
)

type LogWriter struct {
	f *os.File
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	n, err = os.Stderr.Write(p)
	if err != nil {
		return
	}

	return lw.f.Write(p)
}

func init() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	path := filepath.Dir(executable) + "/latest.log"
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatal("{←|⇶} ", err)
	}

	go func() {
		defer f.Close()
		select {}
	}()

	lw := &LogWriter{f}
	log.SetOutput(lw)
}
