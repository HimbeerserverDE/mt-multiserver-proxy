package proxy

import (
	"sync"

	"github.com/anon55555/mt"
)

// A InteractionHandler holds information on how to handle a Minetest Interaction.
type InteractionHandler struct {
	Type        mt.Interaction // can be 255 to register on all interactions
	Handler     func(*ClientConn, *mt.ToSrvInteract) bool
}

var interactionHandlers []InteractionHandler
var interactionHandlerMu sync.RWMutex
var interactionHandlerOnce sync.Once

// RegisterInteractionHandler adds a new InteractionHandler.
func RegisterInteractionHandler(handler InteractionHandler) {
	initInteractionHandlers()

	chatCmdsMu.Lock()
	defer chatCmdsMu.Unlock()

	interactionHandlers = append(interactionHandlers, handler)
}

func initInteractionHandlers() {
	chatCmdsOnce.Do(func() {
		interactionHandlerMu.Lock()
		defer interactionHandlerMu.Unlock()
	
		interactionHandlers = make([]InteractionHandler, 0)
	})
}

func handleInteraction(cmd *mt.ToSrvInteract, cc *ClientConn) bool {
	handled := false
	handle := func(cond bool) {
		if(cond) {
			handled = true
		}
	}

	for _, handler := range interactionHandlers {
		if handler.Type == 255 {
			handle(handler.Handler(cc, cmd))
		}
		if cmd.Action == handler.Type {
			handle(handler.Handler(cc, cmd))
		}
		
	}

	return handled
}
