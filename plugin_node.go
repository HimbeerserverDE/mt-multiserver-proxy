package proxy

import (
	"sync"

	"github.com/anon55555/mt"
)

type NodeHandler struct {
	Node          string
	nodeIds       map[mt.Content]bool
	OnDig         func(*ClientConn, *mt.ToSrvInteract) bool
	OnStopDigging func(*ClientConn, *mt.ToSrvInteract) bool
	OnDug         func(*ClientConn, *mt.ToSrvInteract) bool
	OnPlace       func(*ClientConn, *mt.ToSrvInteract) bool // TODO IMPLEMENTED
}

var nodeHandlers []*NodeHandler
var nodeHandlersMu sync.RWMutex

var mapCache map[[3]int16]*[4096]mt.Content
var mapCacheMu sync.RWMutex
var mapCacheOnce sync.Once

func initMapCache() {
	mapCacheOnce.Do(func() {
		mapCache = map[[3]int16]*[4096]mt.Content{}
	})
}

func RegisterNodeHandler(handler *NodeHandler) {
	nodeHandlersMu.Lock()
	defer nodeHandlersMu.Unlock()

	RegisterCacheNode(handler.Node)
	nodeHandlers = append(nodeHandlers, handler)
}

func initNodeHandlerNodeIds() {
	nodeHandlersMu.RLock()
	defer nodeHandlersMu.RUnlock()

	for _, h := range nodeHandlers {
		if h.nodeIds == nil {
			id := GetNodeId(h.Node)

			if id != nil {
				h.nodeIds = id
			}
		}
	}
}

func GetMapCache() map[[3]int16]*[4096]mt.Content {
	initMapCache()

	mapCacheMu.RLock()
	defer mapCacheMu.RUnlock()

	return mapCache
}

func IsCached(pos [3]int16) bool {
	initMapCache()

	mapCacheMu.RLock()
	defer mapCacheMu.RUnlock()

	blkpos, i := mt.Pos2Blkpos(pos)
	if mapCache[blkpos] == nil {
		return false
	} else {
		return mapCache[blkpos][i] != 0
	}
}

func handleNodeInteraction(cc *ClientConn, pointedNode *mt.PointedNode, cmd *mt.ToSrvInteract) bool {
	nodeHandlersMu.RLock()
	defer nodeHandlersMu.RUnlock()

	mapCacheMu.RLock()
	defer mapCacheMu.RUnlock()

	var handled bool
	for _, handler := range nodeHandlers {
		// check if nodeId is right
		pos, i := mt.Pos2Blkpos(pointedNode.Under)
		if handler.nodeIds[mapCache[pos][i]] {
			var h bool

			switch cmd.Action {
			case mt.Dig:
				if handler.OnDig != nil {
					h = handler.OnDig(cc, cmd)
				}
			case mt.StopDigging:
				if handler.OnStopDigging != nil {
					h = handler.OnStopDigging(cc, cmd)
				}
			case mt.Dug:
				if handler.OnDug != nil {
					h = handler.OnDug(cc, cmd)
				}
			case mt.Place:
				if handler.OnPlace != nil {
					h = handler.OnPlace(cc, cmd)
				}
			}

			if h {
				handled = h
			}
		}
	}

	return handled
}

func initPluginNode() {
	RegisterBlkDataHandler(BlkDataHandler{
		Handler: func(cc *ClientConn, cmd *mt.ToCltBlkData) bool {
			initMapCache()
			initNodeHandlerNodeIds()

			mapCacheMu.Lock()
			defer mapCacheMu.Unlock()

			for i, node := range cmd.Blk.Param0 {
				// check if node is interesting
				interesting := false
				for _, h := range nodeHandlers {
					if h.nodeIds[node] {
						interesting = true
						break
					}
				}

				// if it changed
				if !interesting {
					if mapCache[cmd.Blkpos] != nil {
						if mapCache[cmd.Blkpos][i] != 0 && mapCache[cmd.Blkpos][i] != node {
							interesting = true
						}
					}
				}

				if interesting {
					if mapCache[cmd.Blkpos] == nil {
						mapCache[cmd.Blkpos] = &[4096]mt.Content{}
					}
					mapCache[cmd.Blkpos][i] = node
				}
			}

			return false
		},
	})

	RegisterInteractionHandler(InteractionHandler{
		Type: AnyInteraction,
		Handler: func(cc *ClientConn, cmd *mt.ToSrvInteract) bool {
			handled := false

			if pointedNode, ok := cmd.Pointed.(*mt.PointedNode); ok {
				if IsCached(pointedNode.Under) {
					// is a interesting node
					if handleNodeInteraction(cc, pointedNode, cmd) {
						handled = true
					}
				}
			}

			return handled
		},
	})
}
