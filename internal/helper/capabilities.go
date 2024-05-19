package helper

import (
	"golang.org/x/sys/unix"
)

// Capability represents a Linux capability
type Capability uint32

const (
	// CAP_NET_RAW is the capability to use raw sockets
	CAP_NET_RAW Capability = (1 << uint32(13))
)

// HasCapabilities checks if the current process has the specified capabilities.
func HasCapabilities(cap Capability) bool {
	if unix.Geteuid() == 0 {
		return true
	}

	var hdr unix.CapUserHeader
	var data unix.CapUserData

	hdr.Version = unix.LINUX_CAPABILITY_VERSION_3
	hdr.Pid = 0

	err := unix.Capget(&hdr, &data)
	if err != nil {
		return false
	}

	return !(data.Effective&uint32(cap) == 0)
}
