package proxy

import "sync"

var (
	onChatMsgs  []func(*ClientConn, string) string
	onChatMsgMu sync.RWMutex
)

// RegisterOnChatMsg registers a handler that is called
// when a client sends a chat message that is not a proxy command.
// The returned string overrides the original message.
// Later handlers will receive the modified message.
// Handlers are called in registration order.
// If the final message is empty, it is not forwarded to the upstream server.
func RegisterOnChatMsg(handler func(*ClientConn, string) string) {
	onChatMsgMu.Lock()
	defer onChatMsgMu.Unlock()

	onChatMsgs = append(onChatMsgs, handler)
}
