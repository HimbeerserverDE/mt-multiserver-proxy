package proxy

import (
	"io"
	"sync"
)

// A ChatCmd holds information on how to handle a chat command.
type ChatCmd struct {
	Name        string
	Perm        string
	Help        string
	Usage       string
	TelnetUsage string
	Handler     func(*ClientConn, io.Writer, ...string) string
}

var chatCmds map[string]ChatCmd
var chatCmdsMu sync.RWMutex
var chatCmdsOnce sync.Once

// ChatCmds returns a map of all ChatCmds indexed by their names.
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

// ChatCmdExists reports if a ChatCmd exists.
func ChatCmdExists(name string) bool {
	_, ok := ChatCmds()[name]
	return ok
}

// RegisterChatCmd adds a new ChatCmd. It returns true on success
// and false if a command with the same name already exists.
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
