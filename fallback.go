package proxy

import "github.com/HimbeerserverDE/mt"

func (cc *ClientConn) fallback() bool {
	if cc.fallbackFrom == "" {
		return false
	}

	fallback := config.Servers[cc.fallbackFrom].Fallback
	if fallback == "" {
		ack, _ := cc.SendCmd(cc.whyKicked)

		select {
		case <-cc.Closed():
		case <-ack:
			cc.Close()

			cc.mu.Lock()
			cc.srv = nil
			cc.mu.Unlock()

			if srv := cc.server(); srv != nil {
				srv.mu.Lock()
				srv.clt = nil
				srv.mu.Unlock()
			}
		}

		return true
	}

	// Use HopRaw so the fallback server doesn't get saved
	// as the last server.
	if err := cc.HopRaw(fallback); err != nil {
		cc.Log("<-", "fallback fail:", err)

		ack, _ := cc.SendCmd(&mt.ToCltKick{
			Reason: mt.Custom,
			Custom: "Fallback failed.",
		})

		select {
		case <-cc.Closed():
		case <-ack:
			cc.Close()

			cc.mu.Lock()
			cc.srv = nil
			cc.mu.Unlock()

			if srv := cc.server(); srv != nil {
				srv.mu.Lock()
				srv.clt = nil
				srv.mu.Unlock()
			}
		}
	}

	return true
}
