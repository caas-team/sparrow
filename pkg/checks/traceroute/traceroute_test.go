package traceroute

import (
	"errors"
	"syscall"
	"testing"

	"golang.org/x/net/icmp"
)

func TestTR(t *testing.T) {
	_, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if !errors.Is(err, syscall.EPERM) {
		t.Errorf("err is not eperm")
	}
}
