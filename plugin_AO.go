package proxy

import (
	"github.com/anon55555/mt"
	"sync"
)

type AOHandler struct {
	AOIDs   map[mt.AOID]bool
	OnAOMsg func(*ClientConn, mt.AOID, mt.AOMsg) bool
	OnAOAdd func(*ClientConn, mt.AOID) bool
	OnAORm  func(*ClientConn, mt.AOID) bool
}

var AOHandlers []*AOHandler
var AOHandlersMu sync.RWMutex

func RegisterAOHandler(h *AOHandler) {
	AOHandlersMu.Lock()
	defer AOHandlersMu.Unlock()

	AOHandlers = append(AOHandlers, h)
}

var AOCache map[string]*[]mt.AOInitData
var AOCacheMu sync.RWMutex

func handleAOMsg(sc *ServerConn, id mt.AOID, mg mt.AOMsg) bool {
	for _, handler := range AOHandlers {
		if handler.AOIDs == nil || handler.AOIDs[id] {
			handler.OnAOMsg(sc.clt, id, mg)
		}
	}

	return false
}
