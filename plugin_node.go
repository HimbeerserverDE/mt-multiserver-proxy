package proxy

import (
	"sync"

	"github.com/anon55555/mt"
)

type NodeHandler struct {
	Node          string
	nodeIds       map[mt.Content]bool
	OnDig         func(*ClientConn, *[3]int16) bool
	OnStopDigging func(*ClientConn, *[3]int16) bool
	OnDug         func(*ClientConn, *[3]int16) bool
	OnPlace       func(*ClientConn, *[3]int16) bool
	OnUse         func(*ClientConn, *[3]int16) bool
	OnActivate    func(*ClientConn, *[3]int16) bool
}

var NodeHandlers []*NodeHandler
var NodeHandlersMu sync.RWMutex

var mapCache     map[[3]int16]*[4096]mt.Content
var mapCacheMu   sync.RWMutex
var mapCacheOnce sync.Once

func initMapCache() {
	mapCacheOnce.Do(func() {
		mapCache = map[[3]int16]*[4096]mt.Content{}
	})
}

func RegisterNodeHandler(handler *NodeHandler) {

	NodeHandlersMu.Lock()
	defer NodeHandlersMu.Unlock()

	RegisterNeedNode(handler.Node)
	NodeHandlers = append(NodeHandlers, handler)
}

func initNodeHandlerNodeIds() {
	NodeHandlersMu.RLock()
	defer NodeHandlersMu.RUnlock()
				
	for _, h := range NodeHandlers {
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
	NodeHandlersMu.RLock()
	defer NodeHandlersMu.RUnlock()

	var handled bool

	for _, handler := range NodeHandlers {
		var h bool
	
		switch cmd.Action {
		case mt.Dig:
			if handler.OnDig != nil {
				h = handler.OnDig(cc, &pointedNode.Under)
			}
		case mt.StopDigging:
			if handler.OnStopDigging != nil {
				h = handler.OnStopDigging(cc, &pointedNode.Under)
			}
		case mt.Dug:
			if handler.OnDug != nil {
				h = handler.OnDug(cc, &pointedNode.Under)
			}
		case mt.Place:
			if handler.OnPlace != nil {
				h = handler.OnPlace(cc, &pointedNode.Under)
			}
		case mt.Use:
			if handler.OnUse != nil {
				h = handler.OnUse(cc, &pointedNode.Under)
			}
		case mt.Activate:
			if handler.OnActivate != nil {
				h = handler.OnActivate(cc, &pointedNode.Under)
			}
		}

		if h {
			handled = h
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
				for _, h := range NodeHandlers {
					if h.nodeIds[node] {
						interesting = true
						break
					}
				}

				// if it changed
				// if !interesting && mapCache[cmd.Blkpos][i] != 0 && mapCache[cmd.Blkpos][i] != node {
				if !interesting {
					if mapCache[cmd.Blkpos] != nil {
						if mapCache[cmd.Blkpos][i] != 0 && mapCache[cmd.Blkpos][i] != node {
							interesting = true
						}
					}
				}

				if interesting {
					//cc.Log("<>", "interesting mapBlock")
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

