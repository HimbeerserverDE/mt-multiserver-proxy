package proxy

import (
	"github.com/anon55555/mt"
	"sync"
)

type LeaveType uint8

const (
	Exit LeaveType = iota
	Kick
)

type Leave struct{
	Type LeaveType
	Kick *mt.ToCltKick
}

type PlayerHandler struct{
	Join  func(cc *ClientConn) (destination string)
	Leave func(cc *ClientConn, l *Leave)
	Hop   func(cc *ClientConn, source, destination string)
}

var playerHandlers   []*PlayerHandler
var playerHandlersMu sync.RWMutex

func RegisterPlayerHandler(h *PlayerHandler) {
	playerHandlersMu.Lock()
	defer playerHandlersMu.Unlock()

	playerHandlers = append(playerHandlers, h)
}

func handlePlayerJoin(cc *ClientConn) string {
	playerHandlersMu.RLock()
	defer playerHandlersMu.RUnlock()

	var dest string

	for _, handler := range playerHandlers {
		if handler.Join != nil {
			if d := handler.Join(cc); d != "" {
				dest = d
			}
		}
	}

	return dest
}

func handlePlayerLeave(cc *ClientConn, l *Leave) {
	playerHandlersMu.RLock()
	defer playerHandlersMu.RUnlock()

	for _, handler := range playerHandlers {
		if handler.Leave != nil {
			handler.Leave(cc, l)
		}
	}	
}

func handlePlayerHop(cc *ClientConn, source, leave string) {
	playerHandlersMu.RLock()
	defer playerHandlersMu.RUnlock()

	for _, handler := range playerHandlers {
		if handler.Hop != nil {
			handler.Hop(cc, source, leave)
		}
	}		
}
