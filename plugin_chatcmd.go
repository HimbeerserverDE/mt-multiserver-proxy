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

func ChatCmds() map[string]ChatCmd {
	initChatCmds()

	chatCmdsMu.RLock()
	defer chatCmdsMu.RUnlock()

	cmds := make(map[string]ChatCmd)
	for name, cmd := range chatCmds {
		cmds[name] = cmd
	}

	return cmds
}

func ChatCmdExists(name string) bool {
	cmds := ChatCmds()
	_, ok := cmds[name]
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
