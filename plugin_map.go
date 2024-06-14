package proxy

import (
	"sync"

	"github.com/anon55555/mt"
)

type BlkDataHandler struct {
	UsePos  bool     // if set, filters by BlkPos in Pos
	Pos     [3]int16 // ^ specifies BlkPos when UsePos is set
	Handler func(*ClientConn, *mt.ToCltBlkData) bool
}

var blkDataHandlers []BlkDataHandler
var blkDataHandlersMu sync.RWMutex

var cacheNodes = map[string]map[mt.Content]bool{}
var cacheNodesMu sync.RWMutex

// RegisterCacheNode tells server, that nodename is suppost to be cached
func RegisterCacheNode(nodename string) {
	cacheNodesMu.Lock()
	defer cacheNodesMu.Unlock()

	if cacheNodes[nodename] == nil {
		cacheNodes[nodename] = map[mt.Content]bool{} // default value, empty map
	}
}

// GetNodeId gets the nodeid of a
// If not registerd returns map[mt.Content]bool{}
func GetNodeId(nodename string) map[mt.Content]bool {
	cacheNodesMu.RLock()
	defer cacheNodesMu.RUnlock()

	if cacheNodes[nodename] != nil {
		return cacheNodes[nodename]
	} else {
		return nil
	}
}

// addNodeId sets node id, if allready set, ignore
func addNodeId(nodename string, id mt.Content) {
	cacheNodesMu.Lock()
	defer cacheNodesMu.Unlock()

	if cacheNodes[nodename] != nil {
		cacheNodes[nodename][id] = true
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
		if !handler.UsePos && handler.Handler(cc, cmd) {
			handled = true
		} else if handler.Pos == cmd.Blkpos && handler.Handler(cc, cmd) {
			handled = true
		}
	}

	return handled
}
