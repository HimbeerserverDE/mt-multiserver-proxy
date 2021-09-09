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

func Clts() map[*ClientConn]struct{} {
	clts := make(map[*ClientConn]struct{})
	lm := allListeners()
	for l := range lm {
		for clt := range l.Clts() {
			clts[clt] = struct{}{}
		}
	}

	return clts
}

func Find(name string) *ClientConn {
	for clt := range Clts() {
		if clt.Name() == name {
			return clt
		}
	}

	return nil
}
