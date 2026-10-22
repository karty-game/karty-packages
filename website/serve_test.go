package website

import (
	"context"
	"testing"
)

func TestServeRejectsNonLoopbackAddresses(t *testing.T) {
	for _, address := range []string{"0.0.0.0:8082", ":8082", "[::]:8082", "192.0.2.1:8082", "localhost:8082", "127.0.0.1"} {
		t.Run(address, func(t *testing.T) {
			if err := Serve(context.Background(), t.TempDir(), address); err == nil {
				t.Fatal("accepted a preview address without an explicit loopback IP and port")
			}
		})
	}
}
