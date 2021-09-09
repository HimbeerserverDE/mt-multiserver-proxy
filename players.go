package proxy

import "sync"

var players = make(map[string]struct{})
var playersMu sync.RWMutex

func Players() map[string]struct{} {
	playersMu.RLock()
	defer playersMu.RUnlock()

	p := make(map[string]struct{})
	for player := range players {
		p[player] = struct{}{}
	}

	return p
}
