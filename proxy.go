/*
Package proxy is a minetest reverse proxy for multiple servers.
It also provides an API for plugins.
*/
package proxy

import "regexp"

const (
	latestSerializeVer = 28
	latestProtoVer     = 39
	versionString      = "5.5.0-dev-83a7b48bb"
	maxPlayerNameLen   = 20
	bytesPerMediaBunch = 5000
)

var playerNameChars = regexp.MustCompile("^[a-zA-Z0-9-_]+$")
