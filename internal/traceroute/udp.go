package traceroute

import (
	"context"
	"errors"
	"net"
)

var _ hopper = (*udpHopper)(nil)

type udpHopper struct{ *tracer }

func (h *udpHopper) Hop(_ context.Context, _ *net.IPAddr, _ uint16, _ int) (hop Hop, err error) {
	return hop, errors.New("udp protocol is not supported yet")
}
