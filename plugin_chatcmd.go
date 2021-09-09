package proxy

import "sync"

type ChatCmd struct {
	Name    string
	Perm    string
	Handler func(*ClientConn, ...string) string
}

var chatCmds map[string]ChatCmd
var chatCmdsMu sync.RWMutex
var chatCmdsOnce sync.Once

func ChatCmdExists(name string) bool {
	initChatCmds()

	chatCmdsMu.RLock()
	defer chatCmdsMu.RUnlock()

	_, ok := chatCmds[name]
	return ok
}

func RegisterChatCmd(cmd ChatCmd) bool {
	initChatCmds()

	if ChatCmdExists(cmd.Name) {
		return false
	}

	chatCmdsMu.Lock()
	defer chatCmdsMu.Unlock()

	chatCmds[cmd.Name] = cmd
	return true
}

func initChatCmds() {
	chatCmdsOnce.Do(func() {
		chatCmdsMu.Lock()
		defer chatCmdsMu.Unlock()

		chatCmds = make(map[string]ChatCmd)
	})
}
