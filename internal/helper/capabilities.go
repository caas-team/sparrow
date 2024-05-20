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
func HasCapabilities(ca Capability) bool {
	if unix.Geteuid() == 0 {
		return true
	}

	hdr := &unix.CapUserHeader{
		Pid:     int32(unix.Getpid()),
		Version: unix.LINUX_CAPABILITY_VERSION_3,
	}
	data := &unix.CapUserData{}

	err := unix.Capget(hdr, data)
	if err != nil {
		return false
	}

	return data.Effective&uint32(ca) != 0
}
