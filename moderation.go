package proxy

import (
	"net"

	"github.com/HimbeerserverDE/mt"
)

// Kick sends mt.ToCltKick with the specified custom reason
// and closes the ClientConn.
func (cc *ClientConn) Kick(reason string) {
	go func() {
		ack, _ := cc.SendCmd(&mt.ToCltKick{
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
	ip := cc.RemoteAddr().(*net.UDPAddr).IP.String()
	return DefaultAuth().Ban(ip, cc.name)
}
