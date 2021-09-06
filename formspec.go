package proxy

import (
	"regexp"
	"strings"
)

func (sc *ServerConn) prependFormspec(fs *string) {
	reg := regexp.MustCompile("[^a-zA-Z0-9-_.:]")
	reg2 := regexp.MustCompile("[a-zA-Z0-9-_.]*\\.[a-zA-Z-_.]+")
	subs := reg.Split(*fs, -1)
	seps := reg.FindAllString(*fs, -1)

	for i, sub := range subs {
		if reg2.MatchString(sub) && !strings.Contains(sub, " ") {
			prepend(sc.name, &subs[i])
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
