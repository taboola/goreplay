package rawSocket

import (
	"encoding/binary"
	"log"
	"strconv"
	"strings"
)

var _ = log.Println

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

type tcpID [24]byte

// TCPPacket provides tcp packet parser
// Packet structure: http://en.wikipedia.org/wiki/Transmission_Control_Protocol
type TCPPacket struct {
	SrcPort    uint16
	DestPort   uint16
	Seq        uint32
	Ack        uint32
	OrigAck    uint32
	DataOffset uint8
	IsFIN      bool

	Raw  []byte
	Data []byte
	Addr []byte
	ID   tcpID
}

// ParseTCPPacket takes address and tcp payload and returns parsed TCPPacket
func ParseTCPPacket(addr []byte, data []byte) (p *TCPPacket) {
	p = &TCPPacket{Raw: data}
	p.ParseBasic()
	p.Addr = addr
	p.GenID()

	return
}

func (p *TCPPacket) GenID() {
	copy(p.ID[:16], p.Addr)
	copy(p.ID[16:], p.Raw[0:2])  // Src port
	copy(p.ID[18:], p.Raw[2:4])  // Dest port
	copy(p.ID[20:], p.Raw[8:12]) // Ack
}

func (p *TCPPacket) UpdateAck(ack uint32) {
	p.OrigAck = p.Ack
	p.Ack = ack
	binary.BigEndian.PutUint32(p.Raw[8:12], ack)
	p.GenID()
}

// ParseBasic set of fields
func (t *TCPPacket) ParseBasic() {
	t.DestPort = binary.BigEndian.Uint16(t.Raw[2:4])
	t.SrcPort = binary.BigEndian.Uint16(t.Raw[0:2])
	t.Seq = binary.BigEndian.Uint32(t.Raw[4:8])
	t.Ack = binary.BigEndian.Uint32(t.Raw[8:12])
	t.DataOffset = (t.Raw[12] & 0xF0) >> 4
	t.IsFIN = t.Raw[13]&0x01 != 0

	// log.Println("DataOffset:", t.DataOffset, t.DestPort, t.SrcPort, t.Seq, t.Ack)

	t.Data = t.Raw[t.DataOffset*4:]
}

func (t *TCPPacket) Dump() []byte {
	buf := make([]byte, len(t.Data)+16+16)
	copy(buf[:16], t.Addr)

	tcpBuf := buf[16:]

	binary.BigEndian.PutUint16(tcpBuf[2:4], t.DestPort)
	binary.BigEndian.PutUint16(tcpBuf[0:2], t.SrcPort)

	binary.BigEndian.PutUint32(tcpBuf[4:8], t.Seq)
	binary.BigEndian.PutUint32(tcpBuf[8:12], t.Ack)

	tcpBuf[12] = 64

	if t.IsFIN {
		tcpBuf[13] = tcpBuf[13] | 0x01
	}

	copy(tcpBuf[16:], t.Data)

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
		"FIN:" + strconv.FormatBool(t.IsFIN),

		"Data size:" + strconv.Itoa(len(t.Data)),
		"Data:" + string(t.Data[:maxLen]),
	}, "\n")
}
