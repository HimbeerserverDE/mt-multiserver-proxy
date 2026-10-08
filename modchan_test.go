package proxy

import (
	"testing"
	"time"
)

func TestModChannelWithoutBackendReturnsFailure(t *testing.T) {
	for _, leave := range []bool{false, true} {
		result := make(chan bool, 1)
		go func(leave bool) {
			cc := &ClientConn{}
			if leave {
				result <- <-cc.LeaveModChan("classrooms:cmd")
			} else {
				result <- <-cc.JoinModChan("classrooms:cmd")
			}
		}(leave)
		select {
		case ok := <-result:
			if ok {
				t.Fatal("channel operation succeeded without a backend")
			}
		case <-time.After(time.Second):
			t.Fatalf("channel operation deadlocked (leave=%v)", leave)
		}
	}
}
