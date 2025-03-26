package proxy

import (
	"regexp"
)

var itemName = regexp.MustCompile("(item_image\\[[0-9.-]+,[0-9.-]+;[0-9.-]+,[0-9.-]+;)([a-zA-Z0-9-_.: ]+)(\\])")
var itemButtonName = regexp.MustCompile("(item_image_button\\[[0-9.-]+,[0-9.-]+;[0-9.-]+,[0-9.-]+;)([a-zA-Z0-9-_.: ]+)(;[a-zA-Z0-9-_.: ]+;[^\\[\\]]*\\])")
var textureName = regexp.MustCompile("([a-zA-Z0-9-_.]+\\.(?i:png|jpg|jpeg|bmp|tga|obj|b3d|x|gltf|glb))")

func (sc *ServerConn) prependFormspec(fs *string) {
	*fs = ReplaceAllStringSubmatchFunc(textureName, *fs, func(groups []string) string {
		return sc.mediaPool + "_" + groups[1]
	})
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
