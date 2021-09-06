package proxy

import (
	"strings"
	"sync"

	"github.com/anon55555/mt"
)

type ChatCmd func(*ClientConn, ...string) string

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

func RegisterChatCmd(name string, cmd ChatCmd) bool {
	initChatCmds()

	if ChatCmdExists(name) {
		return false
	}

	chatCmdsMu.Lock()
	defer chatCmdsMu.Unlock()

	chatCmds[name] = cmd
	return true
}

func onChatMsg(cc *ClientConn, cmd *mt.ToSrvChatMsg) string {
	initChatCmds()

	if strings.HasPrefix(cmd.Msg, Conf().CmdPrefix) {
		substrs := strings.Split(cmd.Msg, " ")
		cmdName := strings.Replace(substrs[0], Conf().CmdPrefix, "", 1)

		var args []string
		if len(substrs) > 1 {
			args = substrs[1:]
		}

		v := make([]interface{}, 2+len(args))
		v[0] = "command"
		v[1] = cmdName

		for i, arg := range args {
			v[i+2] = arg
		}

		cc.Log("-->", v...)

		if !ChatCmdExists(cmdName) {
			cc.Log("<--", "unknown command", cmdName)
			return "Command not found."
		}

		chatCmdsMu.RLock()
		defer chatCmdsMu.RUnlock()

		return chatCmds[cmdName](cc, args...)
	}

	return ""
}

func initChatCmds() {
	chatCmdsOnce.Do(func() {
		chatCmdsMu.Lock()
		defer chatCmdsMu.Unlock()

		chatCmds = make(map[string]ChatCmd)
	})
}
