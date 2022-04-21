package proxy

import (
	"crypto/sha1"
	"encoding/base64"
	"os"
	"strings"
)

func (cc *contentConn) fromCache(filename, base64SHA1 string) bool {
	os.Mkdir(Path("cache"), 0777)

	// convert to filename safe b64
	base64SHA1Filesafe := strings.Replace(base64SHA1, "/", "_", -1)
	base64SHA1Filesafe = strings.Replace(base64SHA1Filesafe, "+", "-", -1)

	data, err := os.ReadFile(Path("cache/", base64SHA1Filesafe))
	if err != nil {
		if !os.IsNotExist(err) {
			cc.log("->", "cache", err)
		}

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
		// convert to filename safe b64
		base64SHA1Filesafe := strings.Replace(f.base64SHA1, "/", "_", -1)
		base64SHA1Filesafe = strings.Replace(base64SHA1Filesafe, "+", "-", -1)

		os.WriteFile(Path("cache/", base64SHA1Filesafe), f.data, 0666)
	}
}

func cacheMedia(f mediaFile) {
	hash := sha1.Sum(f.data)
	sum := base64.RawStdEncoding.EncodeToString(hash[:])

	os.WriteFile(Path("cache/", sum), f.data, 0666)
}
