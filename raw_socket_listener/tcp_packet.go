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

// TCPPacket provides tcp packet parser
// Packet structure: http://en.wikipedia.org/wiki/Transmission_Control_Protocol
type TCPPacket struct {
	SrcPort    uint16
	DestPort   uint16
	Seq        uint32
	Ack        uint32
	DataOffset uint8
	Flags      uint16
	Window     uint16
	Checksum   uint16
	Urgent     uint16

	Data []byte

	Addr string
}

// ParseTCPPacket takes address and tcp payload and returns parsed TCPPacket
func ParseTCPPacket(addr string, b []byte) (p *TCPPacket) {
	p = &TCPPacket{Data: b}
	p.ParseBasic()
	p.Addr = addr

	return p
}

// Parse TCP Packet, inspired by: https://github.com/miekg/pcap/blob/master/packet.go
func (t *TCPPacket) Parse() {
	t.ParseBasic()
	t.Flags = binary.BigEndian.Uint16(t.Data[12:14]) & 0x1FF
	t.Window = binary.BigEndian.Uint16(t.Data[14:16])
	t.Checksum = binary.BigEndian.Uint16(t.Data[16:18])
	t.Urgent = binary.BigEndian.Uint16(t.Data[18:20])
}

// ParseBasic set of fields
func (t *TCPPacket) ParseBasic() {
	t.DestPort = binary.BigEndian.Uint16(t.Data[2:4])
	t.SrcPort = binary.BigEndian.Uint16(t.Data[0:2])
	t.Seq = binary.BigEndian.Uint32(t.Data[4:8])
	t.Ack = binary.BigEndian.Uint32(t.Data[8:12])
	t.DataOffset = (t.Data[12] & 0xF0) >> 4

	t.Data = t.Data[t.DataOffset*4:]
}

// String output for a TCP Packet
func (t *TCPPacket) String() string {
	maxLen := len(t.Data)
	if maxLen > 200 {
		maxLen = 200
	}

	return strings.Join([]string{
		"Addr: " + t.Addr,
		"Source port: " + strconv.Itoa(int(t.SrcPort)),
		"Dest port:" + strconv.Itoa(int(t.DestPort)),
		"Sequence:" + strconv.Itoa(int(t.Seq)),
		"Acknowledgment:" + strconv.Itoa(int(t.Ack)),
		"Header len:" + strconv.Itoa(int(t.DataOffset)),

		"Flag ns:" + strconv.FormatBool(t.Flags&fNS != 0),
		"Flag crw:" + strconv.FormatBool(t.Flags&fCWR != 0),
		"Flag ece:" + strconv.FormatBool(t.Flags&fECE != 0),
		"Flag urg:" + strconv.FormatBool(t.Flags&fURG != 0),
		"Flag ack:" + strconv.FormatBool(t.Flags&fACK != 0),
		"Flag psh:" + strconv.FormatBool(t.Flags&fPSH != 0),
		"Flag rst:" + strconv.FormatBool(t.Flags&fRST != 0),
		"Flag syn:" + strconv.FormatBool(t.Flags&fSYN != 0),
		"Flag fin:" + strconv.FormatBool(t.Flags&fFIN != 0),

		"Window size:" + strconv.Itoa(int(t.Window)),
		"Checksum:" + strconv.Itoa(int(t.Checksum)),

		"Data size:" + strconv.Itoa(len(t.Data)),
		"Data:" + string(t.Data[:maxLen]),
	}, "\n")
}
