package rawSocket

import (
	"encoding/binary"
	"strconv"
	"strings"
)

// TCP Flags
const (
	fFIN = 1 << iota
	fSYN
	fRST
	fPSH
	fACK
	fURG
	fECE
	fCWR
	fNS
)

type tcpID [10]byte

// TCPPacket provides tcp packet parser
// Packet structure: http://en.wikipedia.org/wiki/Transmission_Control_Protocol
type TCPPacket struct {
	SrcPort    uint16
	DestPort   uint16
	Seq        uint32
	Ack        uint32
	DataOffset uint8

	Raw []byte
	Data []byte
	Addr []byte
	ID tcpID
}

// ParseTCPPacket takes address and tcp payload and returns parsed TCPPacket
func ParseTCPPacket(addr []byte, data []byte) (p *TCPPacket) {
	p = &TCPPacket{Raw: data}
	p.ParseBasic()
	p.Addr = addr

	copy(p.ID[:4], addr)
	copy(p.ID[4:], p.Raw[2:4]) // Dest port
	copy(p.ID[6:], p.Raw[8:12]) // Ack

	return
}

// ParseBasic set of fields
func (t *TCPPacket) ParseBasic() {
	t.DestPort = binary.BigEndian.Uint16(t.Raw[2:4])
	t.SrcPort = binary.BigEndian.Uint16(t.Raw[0:2])
	t.Seq = binary.BigEndian.Uint32(t.Raw[4:8])
	t.Ack = binary.BigEndian.Uint32(t.Raw[8:12])
	t.DataOffset = (t.Raw[12] & 0xF0) >> 4

	t.Data = t.Raw[t.DataOffset*4:]
}

func (t *TCPPacket) Dump() []byte {
	buf := make([]byte, len(t.Data) + 16 + 4)

	binary.BigEndian.PutUint16(buf[6:8], t.DestPort)
	binary.BigEndian.PutUint16(buf[4:6], t.SrcPort)

	binary.BigEndian.PutUint32(buf[8:12], t.Seq)
	binary.BigEndian.PutUint32(buf[12:16], t.Ack)

	buf[16] = 64
	copy(buf[20:], t.Data)

	return buf
}

// String output for a TCP Packet
func (t *TCPPacket) String() string {
	maxLen := len(t.Data)
	if maxLen > 200 {
		maxLen = 200
	}

	return strings.Join([]string{
		"Addr: " + string(t.Addr),
		"Source port: " + strconv.Itoa(int(t.SrcPort)),
		"Dest port:" + strconv.Itoa(int(t.DestPort)),
		"Sequence:" + strconv.Itoa(int(t.Seq)),
		"Acknowledgment:" + strconv.Itoa(int(t.Ack)),
		"Header len:" + strconv.Itoa(int(t.DataOffset)),

		"Data size:" + strconv.Itoa(len(t.Data)),
		"Data:" + string(t.Data[:maxLen]),
	}, "\n")
}
