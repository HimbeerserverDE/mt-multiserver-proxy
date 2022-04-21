package proxy

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
)

func (cc *contentConn) fromCache(filename, base64SHA1 string) bool {
	os.Mkdir(Path("cache"), 0777)

	hash, err := b64.DecodeString(base64SHA1)
	if err != nil {
		cc.log("<-", base64SHA1, ": ", err)
		return false
	}

	hexSHA1 := hex.EncodeToString(hash)

	data, err := os.ReadFile(Path("cache/", hexSHA1))
	if err != nil {
		return false
	}

	cc.media = append(cc.media, mediaFile{
		name:       filename,
		base64SHA1: base64SHA1,
		data:       data,
	})

	return true
}

func (cc *contentConn) updateCache() {
	os.Mkdir(Path("cache"), 0777)

	for _, f := range cc.media {
		hash, err := b64.DecodeString(f.base64SHA1)
		if err != nil {
			cc.log("<-", f.base64SHA1, ": ", err)
			continue
		}

		hexSHA1 := hex.EncodeToString(hash)
		os.WriteFile(Path("cache/", hexSHA1), f.data, 0666)
	}
}

func cacheMedia(f mediaFile) {
	hash := sha1.Sum(f.data)
	sum := hex.EncodeToString(hash[:])

	os.WriteFile(Path("cache/", sum), f.data, 0666)
}
