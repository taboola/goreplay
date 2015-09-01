package rawSocket

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/buger/gor/proto"
	"log"
	"strconv"
	"sync"
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
	Ack          uint32
	ResponseAck  uint32
	RequestStart int64
	RequestAck   uint32
	Start        int64
	End          int64
	IsIncoming   bool

	packets []*TCPPacket

	timer *time.Timer // Used for expire check

	delChan chan *TCPMessage

	expire *time.Duration

	mu sync.Mutex
}

// NewTCPMessage pointer created from a Acknowledgment number and a channel of messages readuy to be deleted
func NewTCPMessage(ID string, delChan chan *TCPMessage, Ack uint32, expire *time.Duration, IsIncoming bool) (msg *TCPMessage) {
	msg = &TCPMessage{ID: ID, Ack: Ack, expire: expire, IsIncoming: IsIncoming}
	msg.Start = time.Now().UnixNano()
	msg.delChan = delChan // used for notifying that message completed or expired

	return
}

// Timeout notifies message to stop listening, close channel and message ready to be sent
func (t *TCPMessage) Timeout() {
	t.mu.Lock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.mu.Unlock()

	// Notify RAWListener that message is ready to be send to replay server
	// Responses without requests gets discarded
	if t.IsIncoming || t.RequestStart != 0 {
		t.delChan <- t
	}
}

// Bytes return message content
func (t *TCPMessage) Bytes() (output []byte) {
	for _, p := range t.packets {
		output = append(output, p.Data...)
	}

	return output
}

// Size returns total size of message
func (t *TCPMessage) Size() (size int) {
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
		} else {
			t.packets = append([]*TCPPacket{packet}, t.packets...)
		}

		t.End = time.Now().UnixNano()
	}

	if !t.isMultipart() {
		t.Timeout()
	} else {
		t.mu.Lock()
		// If more then 1 packet, wait for more, and set expiration
		if len(t.packets) == 1 {
			// Every time we receive packet we reset this timer
			t.timer = time.AfterFunc(*t.expire, t.Timeout)
		} else {
			// Reset message timeout timer
			if t.timer != nil {
				t.timer.Reset(*t.expire)
			}
		}
		t.mu.Unlock()
	}
}

// isMultipart returns true if message contains from multiple tcp packets
func (t *TCPMessage) isMultipart() bool {
	if len(t.packets) > 1 {
		return true
	}

	payload := t.packets[0].Data
	m := payload[:4]

	if t.IsIncoming {
		// If one GET, OPTIONS, or HEAD request
		if bytes.Equal(m, []byte("GET ")) || bytes.Equal(m, []byte("OPTI")) || bytes.Equal(m, []byte("HEAD")) {
			return false
		} else {
			// Sometimes header comes after the body :(
			if bytes.Equal(m, []byte("POST")) || bytes.Equal(m, []byte("PUT ")) || bytes.Equal(m, []byte("PATC")) {
				if length := proto.Header(payload, []byte("Content-Length")); len(length) > 0 {
					l, _ := strconv.Atoi(string(length))

					// If content-length equal current body length
					if l > 0 && l == len(proto.Body(payload)) {
						return false
					}
				}
			}
		}
	} else {
		if length := proto.Header(payload, []byte("Content-Length")); len(length) > 0 {
			if length[0] == '0' {
				return false
			}

			l, _ := strconv.Atoi(string(length))

			// If content-length equal current body length
			if l > 0 && l == len(proto.Body(payload)) {
				return false
			}
		}
	}

	return true
}

func (t *TCPMessage) UUID() []byte {
	var key []byte

	if t.IsIncoming {
		key = strconv.AppendInt(key, t.Start, 10)
		key = strconv.AppendUint(key, uint64(t.Ack), 10)
	} else {
		key = strconv.AppendInt(key, t.RequestStart, 10)
		key = strconv.AppendUint(key, uint64(t.RequestAck), 10)
	}

	uuid := make([]byte, 40)
	sha := sha1.Sum(key)
	hex.Encode(uuid, sha[:20])

	return uuid
}
