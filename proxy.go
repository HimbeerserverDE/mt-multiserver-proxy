/*
Package proxy is a minetest reverse proxy for multiple servers.
It also provides an API for plugins.
*/
package proxy

const latestSerializeVer = 28
const latestProtoVer = 39
const maxPlayerNameLen = 20
const playerNameChars = "^[a-zA-Z0-9-_]+$"
const bytesPerMediaBunch = 5000
