package proxy

import (
	"regexp"
	"strings"
)

var textureName = regexp.MustCompile("[a-zA-Z0-9-_.]*\\.[a-zA-Z-_.]+")

func (sc *ServerConn) prependFormspec(fs *string) {
	subs := disallowedChars.Split(*fs, -1)
	seps := disallowedChars.FindAllString(*fs, -1)

	for i, sub := range subs {
		if textureName.MatchString(sub) && !strings.Contains(sub, " ") {
			prepend(sc.mediaPool, &subs[i])
		}
	}

	*fs = ""
	for i, sub := range subs {
		*fs += sub
		if i < len(seps) {
			*fs += seps[i]
		}
	}
}
