/*
Package proxy is a minetest reverse proxy for multiple servers.
It also provides an API for plugins.
*/
package proxy

const latestSerializeVer = 28
const latestProtoVer = 39
const versionString = "5.5.0-dev-83a7b48bb"
const maxPlayerNameLen = 20
const playerNameChars = "^[a-zA-Z0-9-_]+$"
const bytesPerMediaBunch = 5000
