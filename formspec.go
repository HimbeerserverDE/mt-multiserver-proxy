package proxy

import (
	"regexp"
	"strings"
)

var itemName = regexp.MustCompile("(item_image\\[[0-9.]+,[0-9.]+;[0-9.]+,[0-9]+;)([a-zA-Z0-9-_.: ]+)(\\])")
var itemButtonName = regexp.MustCompile("(item_image_button\\[[0-9.]+,[0-9.]+;[0-9.]+,[0-9.]+;)([a-zA-Z0-9-_.: ]+)(;[a-zA-Z0-9-_.: ]+;[^\\[\\]]*\\])")
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

	*fs = ReplaceAllStringSubmatchFunc(itemName, *fs, func(groups []string) string {
		return groups[1] + sc.name + "_" + groups[2] + groups[3]
	})
	*fs = ReplaceAllStringSubmatchFunc(itemButtonName, *fs, func(groups []string) string {
		return groups[1] + sc.name + "_" + groups[2] + groups[3]
	})
}

func ReplaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}
