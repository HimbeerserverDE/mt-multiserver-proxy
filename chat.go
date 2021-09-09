package proxy

import (
	"fmt"
	"strings"
	"time"

	"github.com/anon55555/mt"
)

func (cc *ClientConn) SendChatMsg(msg ...string) {
	cc.SendCmd(&mt.ToCltChatMsg{
		Type:      mt.SysMsg,
		Text:      strings.Join(msg, " "),
		Timestamp: time.Now().Unix(),
	})
}

func onChatMsg(cc *ClientConn, cmd *mt.ToSrvChatMsg) (string, bool) {
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
			return "Command not found.", true
		}

		chatCmdsMu.RLock()
		defer chatCmdsMu.RUnlock()

		cmd := chatCmds[cmdName]

		if !cc.HasPerms(cmd.Perm) {
			cc.Log("<--", "deny command", cmdName)
			return fmt.Sprintf("Missing permission %s.", cmd.Perm), true
		}

		return cmd.Handler(cc, args...), true
	}

	return "", false
}
