package proxy

import (
	"sync"

	"github.com/anon55555/mt"
)

type BlkDataHandler struct {
	Pos     [3]int16 // optional TODO: implement
	Handler func(*ClientConn, *mt.ToCltBlkData) bool
}

var blkDataHandlers []BlkDataHandler
var blkDataHandlersMu sync.RWMutex

var neededNodes = map[string]map[mt.Content]bool{}
var neededNodesMu sync.RWMutex

// RegisterNeedNode tells server, that nodename is needed
func RegisterNeedNode(nodename string) {
	neededNodesMu.Lock()
	defer neededNodesMu.Unlock()

	if neededNodes[nodename] == nil {
		neededNodes[nodename] = map[mt.Content]bool{} // default value, empty map
	}
}

// GetNodeId gets the nodeid of a
// If not registerd returns map[mt.Content]bool{}
func GetNodeId(nodename string) map[mt.Content]bool {
	neededNodesMu.RLock()
	defer neededNodesMu.RUnlock()

	if neededNodes[nodename] != nil {
		return neededNodes[nodename]
	} else {
		return nil
	}
}

// addNodeId sets node id, if allready set, ignore
func addNodeId(nodename string, id mt.Content) {
	neededNodesMu.Lock()
	defer neededNodesMu.Unlock()

	if neededNodes[nodename] != nil {
		neededNodes[nodename][id] = true
	}
}

// RegisterBlkDataHandler registers a BlkDataHande
func RegisterBlkDataHandler(handler BlkDataHandler) {
	blkDataHandlersMu.Lock()
	defer blkDataHandlersMu.Unlock()

	blkDataHandlers = append(blkDataHandlers, handler)
}

func handleBlkData(cc *ClientConn, cmd *mt.ToCltBlkData) bool {
	blkDataHandlersMu.RLock()
	defer blkDataHandlersMu.RUnlock()

	handled := false

	for _, handler := range blkDataHandlers {
		if handler.Handler(cc, cmd) {
			handled = true
		}
	}

	return handled
}
