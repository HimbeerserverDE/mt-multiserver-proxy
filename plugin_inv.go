package proxy

import (
	"bytes"
	"sync"

	"github.com/HimbeerserverDE/mt"
)

var (
	onInvAction   []func(*ClientConn, string) string
	onInvActionMu sync.RWMutex
)

// Inv returns a copy of the current server-provided inventory of the client
// or nil if the client is not connected to a server.
func (cc *ClientConn) Inv() mt.Inv {
	sc := cc.server()
	if sc == nil {
		return nil
	}

	var ret mt.Inv

	sb := &bytes.Buffer{}
	sc.inv.Serialize(sb)
	ret.Deserialize(sb)

	return ret
}

// RegisterOnInvAction registers a handler that is called
// when a client attempts to perform an inventory action.
// The returned string overrides the original action.
// Later handlers will receive the modified action.
// Handlers are called in registration order.
// If the final action string is empty, the action is not forwarded
// to the upstream server.
// You may use the mt package to interact with the action strings.
func RegisterOnInvAction(handler func(*ClientConn, string) string) {
	onInvActionMu.Lock()
	defer onInvActionMu.Unlock()

	onInvAction = append(onInvAction, handler)
}
