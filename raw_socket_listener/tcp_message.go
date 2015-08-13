package rawSocket

import (
	"log"
	"sort"
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
	RequestStart int64
	Start        int64
	IsIncoming   bool
	packets      []*TCPPacket

	timer *time.Timer // Used for expire check

	packetsChan chan *TCPPacket

	delChan chan *TCPMessage

	expire *time.Duration
}

// NewTCPMessage pointer created from a Acknowledgment number and a channel of messages readuy to be deleted
func NewTCPMessage(ID string, delChan chan *TCPMessage, Ack uint32, expire *time.Duration, IsIncoming bool) (msg *TCPMessage) {
	msg = &TCPMessage{ID: ID, Ack: Ack, expire: expire, IsIncoming: IsIncoming}
	msg.Start = time.Now().UnixNano()
	msg.packetsChan = make(chan *TCPPacket)
	msg.delChan = delChan // used for notifying that message completed or expired

	// Every time we receive packet we reset this timer
	msg.timer = time.AfterFunc(*msg.expire, msg.Timeout)

	go msg.listen()

	return
}

func (t *TCPMessage) listen() {
	for {
		select {
		case packet, more := <-t.packetsChan:
			if more {
				t.AddPacket(packet)
			} else {
				// Stop loop if channel closed
				return
			}
		}
	}
}

// Timeout notifies message to stop listening, close channel and message ready to be sent
func (t *TCPMessage) Timeout() {
	select {
	// In some cases Timeout can be called multiple times (do not know how yet)
	// Ensure that we did not close channel 2 times
	case packet, ok := <-t.packetsChan:
		if ok {
			t.AddPacket(packet)
		} else {
			return
		}
	default:
		close(t.packetsChan)
		t.delChan <- t // Notify RAWListener that message is ready to be send to replay server
	}
}

// Bytes sorts packets in right orders and return message content
func (t *TCPMessage) Bytes() (output []byte) {
	sort.Sort(sortBySeq(t.packets))

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
		t.packets = append(t.packets, packet)
	}

	// Reset message timeout timer
	t.timer.Reset(*t.expire)
}
