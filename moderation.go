package proxy

import "github.com/anon55555/mt"

// Kick sends mt.ToCltDisco with the specified custom reason
// and closes the ClientConn.
func (cc *ClientConn) Kick(reason string) {
	if reason == "" {
		reason = "Kicked by proxy."
	}

	ack, _ := cc.SendCmd(&mt.ToCltDisco{
		Reason: mt.Custom,
		Custom: reason,
	})

	select {
	case <-cc.Closed():
	case <-ack:
		cc.Close()
	}
}
