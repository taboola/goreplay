package rawSocket

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/buger/gor/proto"
	"log"
	"strconv"
	"time"
)

// TCPMessage ensure that all TCP packets for given request is received, and processed in right sequence
// Its needed because all TCP message can be fragmented or re-transmitted
//
// Each TCP Packet have 2 ids: acknowledgment - message_id, and sequence - packet_id
// Message can be compiled from unique packets with same message_id which sorted by sequence
// Message is received if we didn't receive any packets for 2000ms
type TCPMessage struct {
	ID           string // Message ID
	Seq          uint32
	Ack          uint32
	ResponseAck  uint32
	RequestStart time.Time
	RequestAck   uint32
	RequestID    string
	Start        time.Time
	End          time.Time
	IsIncoming   bool

	packets []*TCPPacket

	delChan chan *TCPMessage
}

// NewTCPMessage pointer created from a Acknowledgment number and a channel of messages readuy to be deleted
func NewTCPMessage(ID string, Seq, Ack uint32, IsIncoming bool) (msg *TCPMessage) {
	msg = &TCPMessage{ID: ID, Seq: Seq, Ack: Ack, IsIncoming: IsIncoming}
	msg.Start = time.Now()

	return
}

// Bytes return message content
func (t *TCPMessage) Bytes() (output []byte) {
	for _, p := range t.packets {
		output = append(output, p.Data...)
	}

	return output
}

// Size returns total body size
func (t *TCPMessage) BodySize() (size int) {
	if len(t.packets) == 0 {
		return 0
	}

	size += len(proto.Body(t.packets[0].Data))

	for _, p := range t.packets[1:] {
		size += len(p.Data)
	}

	return
}

// Size returns total size of message
func (t *TCPMessage) Size() (size int) {
	if len(t.packets) == 0 {
		return 0
	}

	for _, p := range t.packets {
		size += len(p.Data)
	}

	return
}

// AddPacket to the message and ensure packet uniqueness
// TCP allows that packet can be re-send multiple times
func (t *TCPMessage) AddPacket(packet *TCPPacket) {
	packetFound := false

	for _, pkt := range t.packets {
		if packet.Seq == pkt.Seq {
			packetFound = true
			break
		}
	}

	if packetFound {
		log.Println("Received packet with same sequence")
	} else {
		// Packets not always captured in same Seq order, and sometimes we need to prepend
		if len(t.packets) == 0 || packet.Seq > t.packets[len(t.packets)-1].Seq {
			t.packets = append(t.packets, packet)
		} else if packet.Seq < t.packets[0].Seq {
			t.packets = append([]*TCPPacket{packet}, t.packets...)
			t.Seq = packet.Seq // Message Seq should indicated starting seq
		} else { // insert somewhere in the middle...
			for i, p := range t.packets {
				if packet.Seq < p.Seq {
					t.packets = append(t.packets[:i], append([]*TCPPacket{packet}, t.packets[i:]...)...)
					break
				}
			}
		}

		if t.IsIncoming {
			t.End = time.Now()
		} else {
			t.End = time.Now().Add(time.Millisecond)
		}
	}
}

// isMultipart returns true if message contains from multiple tcp packets
func (t *TCPMessage) IsFinished() bool {
	payload := t.packets[0].Data

	if len(payload) < 4 {
		return true
	}

	m := payload[:4]

	if t.IsIncoming {
		// If one GET, OPTIONS, or HEAD request
		if bytes.Equal(m, []byte("GET ")) || bytes.Equal(m, []byte("OPTI")) || bytes.Equal(m, []byte("HEAD")) {
			return true
		} else {
			// Sometimes header comes after the body :(
			if bytes.Equal(m, []byte("POST")) || bytes.Equal(m, []byte("PUT ")) || bytes.Equal(m, []byte("PATC")) {
				if length := proto.Header(payload, []byte("Content-Length")); len(length) > 0 {
					l, _ := strconv.Atoi(string(length))

					// If content-length equal current body length
					if l > 0 && l == t.BodySize() {
						return true
					}
				}
			}
		}
	} else {
		// Request not found
		// Can be because response came first or request request was just missing
		if t.RequestAck == 0 {
			return false
		}

		if length := proto.Header(payload, []byte("Content-Length")); len(length) > 0 {
			if length[0] == '0' {
				return true
			}

			l, _ := strconv.Atoi(string(length))

			// If content-length equal current body length
			if l > 0 && l == t.BodySize() {
				return true
			}
		} else {
			if enc := proto.Header(payload, []byte("Transfer-Encoding")); len(enc) == 0 {
				return true
			}
		}
	}

	return false
}

func (t *TCPMessage) UUID() []byte {
	var key []byte

	if t.IsIncoming {
		key = strconv.AppendInt(key, t.Start.UnixNano(), 10)
		key = strconv.AppendUint(key, uint64(t.Ack), 10)
	} else {
		key = strconv.AppendInt(key, t.RequestStart.UnixNano(), 10)
		key = strconv.AppendUint(key, uint64(t.RequestAck), 10)
	}

	uuid := make([]byte, 40)
	sha := sha1.Sum(key)
	hex.Encode(uuid, sha[:20])

	return uuid
}

func (t *TCPMessage) UpdateResponseAck() uint32 {
	lastPacket := t.packets[len(t.packets)-1]
	t.ResponseAck = lastPacket.Seq + uint32(len(lastPacket.Data))
	return t.ResponseAck
}

func (t *TCPMessage) ResponseID() string {
	return t.packets[0].Addr + strconv.Itoa(int(t.packets[0].SrcPort)) + strconv.Itoa(int(t.ResponseAck))
}
