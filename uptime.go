package proxy

import "time"

var startTime = time.Now()

// Uptime returns the time the proxy has been running for
func Uptime() time.Duration {
	return time.Since(startTime)
}
