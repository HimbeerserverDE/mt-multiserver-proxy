package proxy

import (
	"sync"

	"github.com/HimbeerserverDE/mt"
)

var (
	modChanSubscribers  map[string][]*ClientConn
	modChanSubscriberMu sync.RWMutex
)

var (
	onCltModChanMsg     []func(string, *ClientConn, string) bool
	onCltModChanMsgMu   sync.RWMutex
	onCltModChanMsgOnce sync.Once
)

// SendModChanMsg sends a message to all subscribed clients on a modchannel.
var (
	onSrvModChanMsg     []func(*ClientConn, string, string, string) bool
	onSrvModChanMsgMu   sync.RWMutex
	onSrvModChanMsgOnce sync.Once
)

func SendModChanMsg(channel, msg string) {
	modChanSubscriberMu.RLock()
	defer modChanSubscriberMu.RUnlock()

	subs, _ := modChanSubscribers[channel]
	for _, cc := range subs {
		cc.SendCmd(&mt.ToCltModChanMsg{
			Channel: channel,
			Msg:     msg,
		})
	}
}

// JoinModChan attempts to subscribe to a modchannel, returning a channel
// that yields a boolean indicating success.
// The proxy will try to rejoin the channel after switching servers,
// but this is not guaranteed to succeed and errors will not be reported
// to the caller.
// This condition can be checked against using the IsModChanJoined method.
// This method may block indefinitely if the client switches servers
// before a response is received. If this cannot be controlled,
// using a select statement with a timeout is recommended.
func (cc *ClientConn) JoinModChan(channel string) <-chan bool {
	failCh := make(chan bool)
	failCh <- false

	sc := cc.server()
	if sc == nil {
		return failCh
	}

	successCh := make(chan bool)

	sc.modChanJoinChMu.Lock()
	defer sc.modChanJoinChMu.Unlock()

	if sc.modChanJoinChs[channel] == nil {
		sc.modChanJoinChs[channel] = make(map[chan bool]struct{})
	}
	sc.modChanJoinChs[channel][successCh] = struct{}{}

	sc.SendCmd(&mt.ToSrvJoinModChan{Channel: channel})
	return successCh
}

// LeaveModChan attempts to unscribe from a modchannel, returning a channel
// yielding a boolean indicating success.
func (cc *ClientConn) LeaveModChan(channel string) <-chan bool {
	failCh := make(chan bool)
	failCh <- false

	sc := cc.server()
	if sc == nil {
		return failCh
	}

	successCh := make(chan bool)

	sc.modChanLeaveChMu.Lock()
	defer sc.modChanLeaveChMu.Unlock()

	if sc.modChanLeaveChs[channel] == nil {
		sc.modChanLeaveChs[channel] = make(map[chan bool]struct{})
	}
	sc.modChanLeaveChs[channel][successCh] = struct{}{}

	sc.SendCmd(&mt.ToSrvLeaveModChan{Channel: channel})
	return successCh
}

// IsModChanJoined returns whether this client is currently subscribed
// to the specified modchannel. This is mainly useful for tracking success
// across server hops.
func (cc *ClientConn) IsModChanJoined(channel string) bool {
	cc.modChsMu.RLock()
	defer cc.modChsMu.RUnlock()

	_, ok := cc.modChs[channel]
	return ok
}

// SendModChanMsg sends a message to the current upstream server
// and all subscribed clients connected to it on a modchannel.
func (cc *ClientConn) SendModChanMsg(channel, msg string) bool {
	sc := cc.server()
	if sc == nil {
		return false
	}

	sc.SendCmd(&mt.ToSrvMsgModChan{
		Channel: channel,
		Msg:     msg,
	})

	return true
}

// RegisterOnCltModChanMsg registers a handler that is called
// when a client sends a message on a modchannel.
// If any handler returns true, the message is not forwarded
// to the upstream server.
func RegisterOnCltModChanMsg(handler func(string, *ClientConn, string) bool) {
	onCltModChanMsgMu.Lock()
	defer onCltModChanMsgMu.Unlock()

	onCltModChanMsg = append(onCltModChanMsg, handler)
}

// RegisterOnSrvModChanMsg registers a handler that is called
// when another client of the current upstream server or a server mod
// sends a message on a modchannel.
// If any handler returns true, the message is not forwarded to the client.
func RegisterOnSrvModChanMsg(handler func(*ClientConn, string, string, string) bool) {
	onSrvModChanMsgMu.Lock()
	defer onSrvModChanMsgMu.Unlock()

	onSrvModChanMsg = append(onSrvModChanMsg, handler)
}

func cltLeaveModChan(cc *ClientConn, channel string) {
	modChanSubscriberMu.Lock()
	defer modChanSubscriberMu.Unlock()

	for i, sub := range modChanSubscribers[channel] {
		if sub == cc {
			modChanSubscribers[channel] = append(modChanSubscribers[channel][:i], modChanSubscribers[channel][i+1:]...)
			break
		}
	}
}

func cltLeaveModChans(cc *ClientConn) {
	modChanSubscriberMu.Lock()
	defer modChanSubscriberMu.Unlock()

	for ch := range modChanSubscribers {
		for i, sub := range modChanSubscribers[ch] {
			if sub == cc {
				modChanSubscribers[ch] = append(modChanSubscribers[ch][:i], modChanSubscribers[ch][i+1:]...)
				break
			}
		}
	}
}

func handleCltModChanMsg(cc *ClientConn, cmd *mt.ToSrvMsgModChan) bool {
	onCltModChanMsgMu.RLock()
	defer onCltModChanMsgMu.RUnlock()

	subs, _ := modChanSubscribers[cmd.Channel]
	for _, sub := range subs {
		sub.SendCmd(&mt.ToCltModChanMsg{
			Channel: cmd.Channel,
			Sender:  cc.Name(),
			Msg:     cmd.Msg,
		})
	}

	drop := false
	for _, handler := range onCltModChanMsg {
		if handler(cmd.Channel, cc, cmd.Msg) {
			drop = true
		}
	}

	return drop
}

func handleSrvModChanMsg(cc *ClientConn, cmd *mt.ToCltModChanMsg) bool {
	onSrvModChanMsgMu.RLock()
	defer onSrvModChanMsgMu.RUnlock()

	drop := false
	for _, handler := range onSrvModChanMsg {
		if handler(cc, cmd.Channel, cmd.Sender, cmd.Msg) {
			drop = true
		}
	}

	return drop
}

func init() {
	modChanSubscribers = make(map[string][]*ClientConn)
}
