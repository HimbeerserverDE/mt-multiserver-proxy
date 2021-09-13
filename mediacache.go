package proxy

import (
	"crypto/sha1"
	"os"
	"strings"
)

func (cc *contentConn) fromCache(filename, base64SHA1 string) bool {
	os.Mkdir(Path("cache"), 0777)

	data, err := os.ReadFile(Path("cache/", filename))
	if err != nil {
		return false
	}

	hash := sha1.Sum(data)
	sum := b64.EncodeToString(hash[:])
	if sum != base64SHA1 {
		return false
	}

	cc.media = append(cc.media, mediaFile{
		name:       strings.Replace(filename, cc.name+"_", "", 1),
		base64SHA1: sum,
		data:       data,
	})

	return true
}

func (cc *contentConn) updateCache() {
	os.Mkdir(Path("cache"), 0777)

	for _, f := range cc.media {
		os.WriteFile(Path("cache/", cc.name, "_", f.name), f.data, 0666)
	}
}

func cacheMedia(f mediaFile) {
	os.WriteFile(f.name, f.data, 0666)
}
