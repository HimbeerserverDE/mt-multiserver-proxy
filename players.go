package proxy

import "sync"

var players = make(map[string]struct{})
var playersMu sync.RWMutex

// Players returns the names of all players
// that are currently connected to the proxy.
func Players() map[string]struct{} {
	playersMu.RLock()
	defer playersMu.RUnlock()

	p := make(map[string]struct{})
	for player := range players {
		p[player] = struct{}{}
	}

	return p
}

// Clts returns all ClientConns currently connected to the proxy.
func Clts() map[*ClientConn]struct{} {
	clts := make(map[*ClientConn]struct{})
	lm := allListeners()
	for l := range lm {
		for clt := range l.clients() {
			clts[clt] = struct{}{}
		}
	}

	return clts
}

// Find returns the ClientConn that has the specified player name.
// If no ClientConn is found, nil is returned.
func Find(name string) *ClientConn {
	for clt := range Clts() {
		if clt.Name() == name {
			return clt
		}
	}

	return nil
}
