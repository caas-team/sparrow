package traceroute

import (
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// tcpFlagSYN represents the SYN flag in the TCP header
const tcpFlagSYN = 0x02

// tcpConn represents a TCP connection
type tcpConn struct {
	Conn *net.TCPConn
}

// sendSYN sends a TCP SYN packet
func (t *tcpConn) sendSYN() error {
	syn := &tcpHeader{
		SrcPort: 12345,
		DstPort: 80,
		SeqNum:  0,
		AckNum:  0,
		Flags:   tcpFlagSYN,
		Window:  65535,
	}
	_, err := t.Conn.Write(syn.Marshal())
	return err
}

// tcpHeader represents a TCP header
type tcpHeader struct {
	SrcPort uint16
	DstPort uint16
	SeqNum  uint32
	AckNum  uint32
	Flags   uint8
	Window  uint16
}

// Marshal marshals the TCP header into a byte slice
func (t *tcpHeader) Marshal() []byte {
	b := make([]byte, 20)
	b[0] = byte(t.SrcPort >> 8)
	b[1] = byte(t.SrcPort & 0xff)
	b[2] = byte(t.DstPort >> 8)
	b[3] = byte(t.DstPort & 0xff)
	b[4] = byte(t.SeqNum >> 24)
	b[5] = byte(t.SeqNum >> 16)
	b[6] = byte(t.SeqNum >> 8)
	b[7] = byte(t.SeqNum & 0xff)
	b[8] = byte(t.AckNum >> 24)
	b[9] = byte(t.AckNum >> 16)
	b[10] = byte(t.AckNum >> 8)
	b[11] = byte(t.AckNum & 0xff)
	b[12] = 0x50 // Data offset
	b[13] = t.Flags
	b[14] = byte(t.Window >> 8)
	b[15] = byte(t.Window & 0xff)
	b[16] = 0 // Checksum placeholder
	b[17] = 0 // Checksum placeholder
	b[18] = 0 // Urgent pointer
	b[19] = 0 // Urgent pointer
	return b
}

func (t *tracer) hopTCP(destAddr *net.IPAddr, ttl int) (hop Hop, err error) {
	// network := "ip4:tcp"
	icmpConn, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		return hop, fmt.Errorf("error creating ICMP listener: %w", err)
	}
	defer func() {
		if cErr := icmpConn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: destAddr.IP, Port: 80})
	if err != nil {
		return hop, fmt.Errorf("error creating TCP connection: %w", err)
	}
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			err = errors.Join(err, ErrClosingConn{Err: cErr})
		}
	}()

	start := time.Now()
	ipConn := ipv4.NewConn(conn)
	if err := ipConn.SetTTL(ttl); err != nil {
		return hop, fmt.Errorf("error setting TTL: %w", err)
	}

	// Send TCP SYN packet
	tcpConn := &tcpConn{
		Conn: conn,
	}
	if err := tcpConn.sendSYN(); err != nil {
		return hop, fmt.Errorf("error sending TCP SYN packet: %w", err)
	}

	recvBuffer := make([]byte, bufferSize)
	icmpConn.SetReadDeadline(time.Now().Add(t.Timeout))

	hop, err = receiveICMPResponse(icmpConn, recvBuffer, start)
	hop.Tracepoint = ttl
	if err != nil {
		return hop, err
	}

	return hop, nil
}
