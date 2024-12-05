package proxy

import (
	"strings"
	"sync"

	"github.com/HimbeerserverDE/mt"
)

var (
	onPlayerReceiveFields     map[string][]func(*ClientConn, []mt.Field)
	onPlayerReceiveFieldsMu   sync.Mutex
	onPlayerReceiveFieldsOnce sync.Once
)

// ShowFormspec opens a formspec on the client.
// The form name should follow the standard minetest naming convention,
// i.e. "pluginname:formname".
// If formname is empty, it reshows the inventory formspec
// without updating it for future opens.
// If formspec is empty, the formspec is closed.
// If formname is empty at the same time, any formspec is closed
// regardless of its name.
func (cc *ClientConn) ShowFormspec(formname string, formspec string) {
	cc.SendCmd(&mt.ToCltShowFormspec{
		Formspec: cc.FormspecPrepend + formspec,
		Formname: formname,
	})
}

// FormspecEscape escapes characters that cannot be used in formspecs.
func FormspecEscape(formspec string) string {
	formspec = strings.ReplaceAll(formspec, "\\", "\\\\")
	formspec = strings.ReplaceAll(formspec, "[", "\\[")
	formspec = strings.ReplaceAll(formspec, "]", "\\]")
	formspec = strings.ReplaceAll(formspec, ";", "\\;")
	formspec = strings.ReplaceAll(formspec, ",", "\\,")
	formspec = strings.ReplaceAll(formspec, "$", "\\$")
	return formspec
}

// RegisterOnPlayerReceiveFields registers a callback that is called
// when a client submits a specific formspec.
// Events triggering this callback include a button being pressed,
// Enter being pressed while a text field is focused,
// a checkbox being toggled, an item being selected in a dropdown list,
// selecting a new tab, changing the selection in a textlist or table,
// selecting an entry in a textlist or table, a scrollbar being moved
// or the form actively being closed by the player.
func RegisterOnPlayerReceiveFields(formname string, handler func(*ClientConn, []mt.Field)) {
	initOnPlayerReceiveFields()

	onPlayerReceiveFieldsMu.Lock()
	defer onPlayerReceiveFieldsMu.Unlock()

	onPlayerReceiveFields[formname] = append(onPlayerReceiveFields[formname], handler)
}

func handleOnPlayerReceiveFields(cc *ClientConn, cmd *mt.ToSrvInvFields) bool {
	onPlayerReceiveFieldsMu.Lock()
	defer onPlayerReceiveFieldsMu.Unlock()

	handlers, ok := onPlayerReceiveFields[cmd.Formname]
	if !ok || len(handlers) == 0 {
		return false
	}

	for _, handler := range handlers {
		handler(cc, cmd.Fields)
	}

	return true
}

func initOnPlayerReceiveFields() {
	onPlayerReceiveFieldsOnce.Do(func() {
		onPlayerReceiveFieldsMu.Lock()
		defer onPlayerReceiveFieldsMu.Unlock()

		onPlayerReceiveFields = make(map[string][]func(*ClientConn, []mt.Field))
	})
}
