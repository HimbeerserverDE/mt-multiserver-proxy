package proxy

import "github.com/anon55555/mt"

// Kick sends mt.ToCltDisco with the specified custom reason
// and closes the ClientConn.
func (cc *ClientConn) Kick(reason string) {
	go func() {
		ack, _ := cc.SendCmd(&mt.ToCltDisco{
			Reason: mt.Custom,
			Custom: reason,
		})

		select {
		case <-cc.Closed():
		case <-ack:
			cc.Close()
		}
	}()
}

// Ban disconnects the ClientConn and prevents the underlying
// network address from connecting again.
func (cc *ClientConn) Ban() error {
	cc.Kick("Banned by proxy.")
	return authIface.Ban(cc.RemoteAddr().String(), cc.name)
}

// Unban removes a player from the ban list. It accepts both
// network addresses and player names.
func Unban(id string) error {
	return authIface.Unban(id)
}
