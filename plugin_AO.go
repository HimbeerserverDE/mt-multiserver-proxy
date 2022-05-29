package proxy

import (
	"github.com/anon55555/mt"
	"sync"
)

type AOHandler struct {
	AOIDs map[string]map[mt.AOID]bool

	OnAOMsg func(*ClientConn, mt.AOID, mt.AOMsg) bool
	OnAOAdd func(*ClientConn, mt.AOID, *mt.AOAdd) bool
	OnAORm  func(*ClientConn, mt.AOID) bool
}

// TODO:
// var aOCache   map[mt.AOID]*mt.AOProps
// var aOCacheMu sync.RWMutex

var aOHandlers []*AOHandler
var aOHandlersMu sync.RWMutex

func handleAOAdd(sc *ServerConn, id mt.AOID, msg *mt.AOAdd) bool {
	var handled bool

	for _, handler := range aOHandlers {
		if handler.OnAOAdd == nil {
			continue
		}
		if handler.AOIDs == nil && handler.OnAOAdd(sc.clt, id, msg) {
			handled = true
		} else if handler.AOIDs[sc.name][id] && handler.OnAOAdd(sc.clt, id, msg) {
			handled = true
		}
	}

	return handled
}

func handleAORm(sc *ServerConn, id mt.AOID) bool {
	var handled bool

	for _, handler := range aOHandlers {
		if handler.OnAORm == nil {
			continue
		}
		if handler.AOIDs == nil && handler.OnAORm(sc.clt, id) {
			handled = true
		} else if handler.AOIDs[sc.name][id] && handler.OnAORm(sc.clt, id) {
			handled = true
		}
	}

	return handled
}

func handleAOMsg(sc *ServerConn, id mt.AOID, msg mt.AOMsg) bool {
	var handled bool

	for _, handler := range aOHandlers {
		if handler.OnAOMsg == nil {
			continue
		}
		if handler.AOIDs == nil && handler.OnAOMsg(sc.clt, id, msg) {
			handled = true
		} else if handler.AOIDs[sc.name][id] && handler.OnAOMsg(sc.clt, id, msg) {
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

///
/// - AOID managment
///

var LowestAOID mt.AOID = 0xA000

///
/// - Global AOIDs
///

type globalAOID struct {
	used, client, server, global bool
}

var globalAOIDs = make(map[mt.AOID]globalAOID)
var globalAOIDsMu sync.RWMutex

// GetGlobalAOID returns a globally unused AOID
func GetGlobalAOID() (bool, mt.AOID) {
	globalAOIDsMu.Lock()
	defer globalAOIDsMu.Unlock()

	notFound := true
	var id mt.AOID = 0xFFFF
	for notFound {
		if globalAOIDs[id].used == false {
			notFound = false

			globalAOIDs[id] = globalAOID{used: true, global: true}
		} else if id < LowestAOID {
			return false, 0xFFFF
		} else {
			id--
		}
	}

	return !notFound, id
}

// FreeAOID marks a id as globally unused
func FreeGlobalAOID(id mt.AOID, srv string) {
	globalAOIDsMu.Lock()
	defer globalAOIDsMu.Unlock()

	globalAOIDs[id] = globalAOID{used: false}
}

///
/// - Server AOIDs
///

type sAOID map[mt.AOID]bool

var serverAOIDs = make(map[string]sAOID)
var serverAOIDsMu sync.RWMutex

func (sm sAOID) empty() bool {
	for _, v := range sm {
		if v {
			return false
		}
	}

	return true
}

// GetServerAOId returns a free mt.AOID for a server, which will be marked as used
func GetServerAOId(srv string) (bool, mt.AOID) {
	globalAOIDsMu.Lock()
	defer globalAOIDsMu.Unlock()

	serverAOIDsMu.Lock()
	defer serverAOIDsMu.Unlock()

	notFound := true
	var id mt.AOID = 0xFFFF
	for notFound {
		if globalAOIDs[id].used == false || (globalAOIDs[id].server == true && !serverAOIDs[srv][id]) {
			notFound = false
			if serverAOIDs[srv] == nil {
				serverAOIDs[srv] = make(map[mt.AOID]bool)
			}
			serverAOIDs[srv][id] = true
			globalAOIDs[id] = globalAOID{used: true, server: true}
		} else if id < LowestAOID {
			return false, 0xFFFF
		} else {
			id--
		}
	}

	return !notFound, id
}

// FreeServerAOID frees a server AOID
func FreeServerAOID(srv string, id mt.AOID) {
	globalAOIDsMu.Lock()
	defer globalAOIDsMu.Unlock()

	serverAOIDsMu.Lock()
	defer serverAOIDsMu.Unlock()

	serverAOIDs[srv][id] = false
	if serverAOIDs[srv].empty() {
		globalAOIDs[id] = globalAOID{used: false}
	}
}

///
/// - clientbound AOIDs
///

type cAOID map[mt.AOID]bool

var clientAOIDs = make(map[string]cAOID)
var clientAOIDsMu sync.RWMutex

func (ca cAOID) empty() bool {
	for _, v := range ca {
		if v {
			return false
		}
	}

	return true
}

// GetFreeAOID returns the next free AOID for a client
func (cc *ClientConn) GetFreeAOID() (bool, mt.AOID) {
	globalAOIDsMu.Lock()
	defer globalAOIDsMu.Unlock()

	clientAOIDsMu.Lock()
	defer clientAOIDsMu.Unlock()

	name := cc.Name()

	notFound := true
	var id mt.AOID = 0xFFFF
	for notFound {
		if globalAOIDs[id].used == false || (globalAOIDs[id].client == true && !clientAOIDs[name][id]) {
			notFound = false
			if clientAOIDs[name] == nil {
				clientAOIDs[name] = make(map[mt.AOID]bool)
			}
			clientAOIDs[name][id] = true
			globalAOIDs[id] = globalAOID{used: true, client: true}
		} else if id < LowestAOID {
			return false, 0xFFFF
		} else {
			id--
		}
	}

	return !notFound, id
}

// FreeAOID marks AOID as free
func (cc *ClientConn) FreeAOID(id mt.AOID) {
	globalAOIDsMu.Lock()
	defer globalAOIDsMu.Unlock()

	clientAOIDsMu.Lock()
	defer clientAOIDsMu.Unlock()

	name := cc.Name()
	clientAOIDs[name][id] = false
	if clientAOIDs[name].empty() {
		globalAOIDs[id] = globalAOID{used: false}
	}
}
