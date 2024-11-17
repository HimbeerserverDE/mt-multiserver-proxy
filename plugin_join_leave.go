package proxy

import "sync"

var (
	onJoin     []func(*ClientConn) string
	onJoinMu   sync.Mutex
	onJoinOnce sync.Once
)

var (
	onLeave     []func(*ClientConn)
	onLeaveMu   sync.Mutex
	onLeaveOnce sync.Once
)

// RegisterOnJoin registers a handler that is called
// when a client finishes connecting to the proxy (TOSERVER_CLIENT_READY packet)
// but before it is connected to an upstream server.
// If any handler returns a non-empty string, the client is kicked
// with that message.
// Handlers are run sequentially and block the client's connection
// and packet handling procedure.
func RegisterOnJoin(handler func(*ClientConn) string) {
	initOnJoin()

	onJoinMu.Lock()
	defer onJoinMu.Unlock()

	onJoin = append(onJoin, handler)
}

// RegisterOnLeave registers a handler that is called
// when a client disconnects for any reason after reaching the ready stage.
// Handlers are run sequentially.
func RegisterOnLeave(handler func(*ClientConn)) {
	initOnLeave()

	onLeaveMu.Lock()
	defer onLeaveMu.Unlock()

	onLeave = append(onLeave, handler)
}

func handleJoin(cc *ClientConn) {
	onJoinMu.Lock()
	defer onJoinMu.Unlock()

	for _, handler := range onJoin {
		if msg := handler(cc); msg != "" {
			cc.Kick(msg)
			break
		}
	}
}

func handleLeave(cc *ClientConn) {
	onLeaveMu.Lock()
	defer onLeaveMu.Unlock()

	for _, handler := range onLeave {
		handler(cc)
	}
}

func initOnJoin() {
	onJoinOnce.Do(func() {
		onJoinMu.Lock()
		defer onJoinMu.Unlock()

		onJoin = make([]func(*ClientConn) string, 0)
	})
}

func initOnLeave() {
	onLeaveOnce.Do(func() {
		onLeaveMu.Lock()
		defer onLeaveMu.Unlock()

		onLeave = make([]func(*ClientConn), 0)
	})
}
