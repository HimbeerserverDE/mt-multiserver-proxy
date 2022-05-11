package proxy

import (
	"sync"

	"github.com/anon55555/mt"
)

// A InteractionHandler holds information on how to handle a Minetest Interaction.
type InteractionHandler struct {
	Type    Interaction
	Handler func(*ClientConn, *mt.ToSrvInteract) bool
}

type Interaction uint8

const (
	Dig Interaction = iota
	StopDigging
	Dug
	Place
	Use
	Activate
	AnyInteraction = 255
)

var interactionHandlers []InteractionHandler
var interactionHandlerMu sync.RWMutex
var interactionHandlerOnce sync.Once

// RegisterInteractionHandler adds a new InteractionHandler.
func RegisterInteractionHandler(handler InteractionHandler) {
	interactionHandlerMu.Lock()
	defer interactionHandlerMu.Unlock()

	interactionHandlers = append(interactionHandlers, handler)
}

func handleInteraction(cmd *mt.ToSrvInteract, cc *ClientConn) bool {
	handled := false

	for _, handler := range interactionHandlers {
		interaction := Interaction(handler.Type)
		if interaction == AnyInteraction || interaction == handler.Type {
			if handler.Handler(cc, cmd) {
				handled = true
			}
		}
	}

	return handled
}
