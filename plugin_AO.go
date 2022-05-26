package proxy

import (
	"github.com/anon55555/mt"
	"sync"
)

type AOHandler struct {
	AOIDs map[mt.AOID]bool

	OnAOMsg func(*ClientConn, mt.AOID, mt.AOMsg) bool
	OnAOAdd func(*ClientConn, mt.AOID, *mt.AOAdd) bool
	OnAORm  func(*ClientConn, mt.AOID) bool
}

// TODO:
// var aOCache   map[mt.AOID]*mt.AOProps
// var aOCacheMu sync.RWMutex

var aOHandlers []*AOHandler
var aOHandlersMu sync.RWMutex

func handleAOMsg(sc *ServerConn, id mt.AOID, msg mt.AOMsg) bool {
	var handled bool

	for _, handler := range aOHandlers {
		if handler.AOIDs == nil && handler.OnAOMsg(sc.clt, id, msg) {
			handled = true
		} else if handler.AOIDs[id] && handler.OnAOMsg(sc.clt, id, msg) {
			handled = true
		}
	}

	return handled
}

func RegisterAOHandler(h *AOHandler) {
	aOHandlersMu.Lock()
	defer aOHandlersMu.Unlock()

	aOHandlers = append(aOHandlers, h)
}
