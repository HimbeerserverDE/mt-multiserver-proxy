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

// RegisterOnCltModChanMsg registers a handler that is called
// when a client sends a message on a modchannel.
// If any handler returns true, the message is not forwarded
// to the upstream server.
func RegisterOnCltModChanMsg(handler func(string, *ClientConn, string) bool) {
	onCltModChanMsgMu.Lock()
	defer onCltModChanMsgMu.Unlock()

	onCltModChanMsg = append(onCltModChanMsg, handler)
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

func init() {
	modChanSubscribers = make(map[string][]*ClientConn)
}
