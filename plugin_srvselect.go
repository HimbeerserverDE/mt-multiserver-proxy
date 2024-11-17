package proxy

import (
	"errors"
	"sync"
)

var (
	ErrEmptySrvSelectorName = errors.New("server selector name is empty")
)

var (
	srvSelectors    map[string]func(*ClientConn) (string, Server)
	srvSelectorMu   sync.RWMutex
	srvSelectorOnce sync.Once
)

// RegisterSrvSelector registers a server selection handler
// which can be enabled using the `SrvSelector` config option.
// Empty names are an error.
// If the handler returns an empty server name,
// the regular server selection procedure is used.
// Only one server selector can be active at a time.
func RegisterSrvSelector(name string, sel func(*ClientConn) (string, Server)) error {
	initSrvSelectors()

	if name == "" {
		return ErrEmptySrvSelectorName
	}

	srvSelectorMu.Lock()
	defer srvSelectorMu.Unlock()

	srvSelectors[name] = sel
	return nil
}

func selectSrv(cc *ClientConn) (string, Server) {
	sel := Conf().SrvSelector

	if sel == "" {
		return "", Server{}
	}

	initSrvSelectors()

	srvSelectorMu.RLock()
	defer srvSelectorMu.RUnlock()

	handler, ok := srvSelectors[sel]
	if !ok {
		cc.Log("<-", "server selector not registered")
		return "", Server{}
	}

	return handler(cc)
}

func initSrvSelectors() {
	srvSelectorOnce.Do(func() {
		srvSelectorMu.Lock()
		defer srvSelectorMu.Unlock()

		srvSelectors = make(map[string]func(*ClientConn) (string, Server))
	})
}
