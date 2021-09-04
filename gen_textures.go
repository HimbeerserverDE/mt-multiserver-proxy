//go:build ignore

// This program generates default_textures.go. It can be invoked
// by running go generate
package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

var b64 = base64.StdEncoding

func main() {
	f, err := os.OpenFile("default_textures.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString("package main\n")
	f.WriteString("\n")
	f.WriteString("var defaultTextures = []mediaFile{\n")

	dir, err := os.ReadDir("textures")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, file := range dir {
		name := file.Name()
		data, err := os.ReadFile("textures/" + name)
		if err != nil {
			fmt.Println(err)
			continue
		}

		sum := sha1.Sum(data)

		f.WriteString("\tmediaFile{\n")
		f.WriteString("\t\tname:       \"" + name + "\",\n")
		f.WriteString("\t\tbase64SHA1: \"" + b64.EncodeToString(sum[:]) + "\",\n")
		f.WriteString("\t\tdata:       []byte{")

		var strs []string
		for _, b := range data {
			strs = append(strs, "0x"+hex.EncodeToString([]byte{b}))
		}

		f.WriteString(strings.Join(strs, ", ") + "},\n")
		f.WriteString("\t},\n")
	}

	f.WriteString("}\n")
}
