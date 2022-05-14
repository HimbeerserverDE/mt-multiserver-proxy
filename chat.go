package proxy

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/anon55555/mt"
)

// ChatCmdTimeout is the time needed until a user is warned
// about a chat command that's taking long to execute.
var ChatCmdTimeout = 10 * time.Second

// DoChatMsg handles a chat message string
// as if it was sent by the ClientConn.
func (cc *ClientConn) DoChatMsg(msg string) {
	cmd := &mt.ToSrvChatMsg{Msg: msg}

	result, isCmd := onChatMsg(cc, cmd)
	if result != "" {
		cc.SendChatMsg(result)
	}

	if !isCmd {
		cc.server().SendCmd(cmd)
	}
}

// SendChatMsg sends a chat message to the ClientConn.
func (cc *ClientConn) SendChatMsg(msg ...string) {
	cc.SendCmd(&mt.ToCltChatMsg{
		Type:      mt.SysMsg,
		Text:      strings.Join(msg, " "),
		Timestamp: time.Now().Unix(),
	})
}

// Colorize returns the minetest-colorized version of the input.
func Colorize(text, color string) string {
	return string([]rune{0x1b}) + "(c@" + color + ")" + text + string([]rune{0x1b}) + "(c@#FFF)"
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

		cc.Log("->", v...)

		if !ChatCmdExists(cmdName) {
			cc.Log("<-", "unknown command", cmdName)
			return "Command not found.", true
		}

		chatCmdsMu.RLock()
		defer chatCmdsMu.RUnlock()

		cmd := chatCmds[cmdName]

		if !cc.HasPerms(cmd.Perm) {
			cc.Log("<-", "deny command", cmdName)
			return fmt.Sprintf("Missing permission %s.", cmd.Perm), true
		}

		return cmd.Handler(cc, nil, args...), true
	}

	return "", false
}

func onTelnetMsg(tlog func(dir string, v ...interface{}), w io.Writer, msg string) string {
	initChatCmds()

	substrs := strings.Split(msg, " ")
	cmdName := substrs[0]

	var args []string
	if len(substrs) > 1 {
		args = substrs[1:]
	}

	if !ChatCmdExists(cmdName) {
		tlog("<-", "unknown command", cmdName)
		return "Command not found.\n"
	}

	chatCmdsMu.RLock()
	defer chatCmdsMu.RUnlock()

	cmd := chatCmds[cmdName]
	return cmd.Handler(nil, w, args...) + "\n"
}
