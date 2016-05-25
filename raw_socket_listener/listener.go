/*
Package rawSocket provides traffic sniffier using RAW sockets.

Capture traffic from socket using RAW_SOCKET's
http://en.wikipedia.org/wiki/Raw_socket

RAW_SOCKET allow you listen for traffic on any port (e.g. sniffing) because they operate on IP level.

Ports is TCP feature, same as flow control, reliable transmission and etc.

This package implements own TCP layer: TCP packets is parsed using tcp_packet.go, and flow control is managed by tcp_message.go
*/
package rawSocket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

var _ = fmt.Println

type wrongCloser interface {
	Close()
}
type wrongCloserProxy struct {
	w wrongCloser
}
func (c wrongCloserProxy) Close() error {
	c.w.Close()
	return nil
}

// Listener handle traffic capture
type Listener struct {
	mu sync.Mutex
	// buffer of TCPMessages waiting to be send
	// ID -> TCPMessage
	messages map[tcpID]*TCPMessage

	// Expect: 100-continue request is send in 2 tcp messages
	// We store ACK aliases to merge this packets together
	ackAliases map[uint32]uint32
	// To get ACK of second message we need to compute its Seq and wait for them message
	seqWithData map[uint32]uint32

	// Ack -> Req
	respAliases map[uint32]*TCPMessage

	// Ack -> ID
	respWithoutReq map[uint32]tcpID

	// Messages ready to be send to client
	packetsChan chan []byte

	// Messages ready to be send to client
	messagesChan chan *TCPMessage

	addr string // IP to listen
	port uint16 // Port to listen

	trackResponse bool
	messageExpire time.Duration

	connHandles []io.Closer

	quit    chan bool
	readyCh chan bool
}

type request struct {
	id    tcpID
	start time.Time
	ack   uint32
}

// Available engines for intercepting traffic
const (
	EngineRawSocket = 1 << iota
	EnginePcap
)

// NewListener creates and initializes new Listener object
func NewListener(addr string, port string, engine int, trackResponse bool, expire time.Duration) (l *Listener) {
	l = &Listener{}

	l.packetsChan = make(chan []byte, 10000)
	l.messagesChan = make(chan *TCPMessage, 10000)
	l.quit = make(chan bool)
	l.readyCh = make(chan bool, 1)

	l.messages = make(map[tcpID]*TCPMessage)
	l.ackAliases = make(map[uint32]uint32)
	l.seqWithData = make(map[uint32]uint32)
	l.respAliases = make(map[uint32]*TCPMessage)
	l.respWithoutReq = make(map[uint32]tcpID)
	l.trackResponse = trackResponse

	l.addr = addr
	_port, _ := strconv.Atoi(port)
	l.port = uint16(_port)

	if expire.Nanoseconds() == 0 {
		expire = 2000 * time.Millisecond
	}

	l.messageExpire = expire

	go l.listen()

	// Special case for testing
	if l.port != 0 {
		if runtime.GOOS == "windows" {
			engine = EngineRawSocket
		}

		switch engine {
		case EngineRawSocket:
			go l.readRAWSocket()
		case EnginePcap:
			go l.readPcap()
		default:
			log.Fatal("Unknown traffic interception engine:", engine)
		}
	}

	return
}

func (t *Listener) listen() {
	gcTicker := time.Tick(t.messageExpire / 2)

	for {
		select {
		case <-t.quit:
			t.Close()
			return
		case data := <-t.packetsChan:
			packet := ParseTCPPacket(data[:16], data[16:])
			t.processTCPPacket(packet)
		case <-gcTicker:
			now := time.Now()

			// Dispatch requests before responses
			for _, message := range t.messages {
				if now.Sub(message.End) >= t.messageExpire {
					t.dispatchMessage(message)
				}
			}
		}
	}
}

func (t *Listener) deleteMessage(message *TCPMessage) {
	delete(t.messages, message.ID())
	delete(t.ackAliases, message.Ack)
	if message.DataAck != 0 {
		delete(t.ackAliases, message.DataAck)
	}
	if message.DataSeq != 0 {
		delete(t.seqWithData, message.DataSeq)
	}

	delete(t.respAliases, message.ResponseAck)
}

func (t *Listener) dispatchMessage(message *TCPMessage) {
	// If already dispatched
	if _, ok := t.messages[message.ID()]; !ok {
		return
	}

	t.deleteMessage(message)

	// log.Println("Dispatching, message", message.Start.UnixNano(), message.Seq, message.Ack, string(message.Bytes()))

	if message.IsIncoming {
		// If there were response before request
		// log.Println("Looking for Response: ", t.respWithoutReq, message.ResponseAck)
		if t.trackResponse {
			if respID, ok := t.respWithoutReq[message.ResponseAck]; ok {
				if resp, rok := t.messages[respID]; rok {
					// if resp.AssocMessage == nil {
					// log.Println("FOUND RESPONSE")
					resp.AssocMessage = message
					message.AssocMessage = resp

					if resp.IsFinished() {
						defer t.dispatchMessage(resp)
					}
					// }
				}
			}

			if resp, ok := t.messages[message.ResponseID]; ok {
				resp.AssocMessage = message
			}
		}
	} else {
		if message.AssocMessage == nil {
			if responseRequest, ok := t.respAliases[message.Ack]; ok {
				message.AssocMessage = responseRequest
				responseRequest.AssocMessage = message
			}
		}

		delete(t.respAliases, message.Ack)
		delete(t.respWithoutReq, message.Ack)

		// Do not track responses which have no associated requests
		if message.AssocMessage == nil {
			// log.Println("Can't dispatch resp", message.Seq, message.Ack, string(message.Bytes()))
			return
		}
	}

	t.messagesChan <- message
}

func (t *Listener) readRAWSocket() {
	conn, e := net.ListenPacket("ip:tcp", t.addr)
	t.connHandles = append(t.connHandles, conn)

	if e != nil {
		log.Fatal(e)
	}

	defer conn.Close()

	buf := make([]byte, 64*1024) // 64kb

	t.readyCh <- true

	for {
		// Note: ReadFrom receive messages without IP header
		n, addr, err := conn.ReadFrom(buf)

		if err != nil {
			if strings.HasSuffix(err.Error(), "closed network connection") {
				return
			} else {
				continue
			}
		}

		if n > 0 {
			if t.isValidPacket(buf[:n]) {
				newBuf := make([]byte, n+16)
				copy(newBuf[16:], buf[:n])
				copy(newBuf[:16], []byte(addr.(*net.IPAddr).IP))

				t.packetsChan <- newBuf
			}
		}
	}
}

func (t *Listener) isValidPacket(buf []byte) bool {
	// To avoid full packet parsing every time, we manually parsing values needed for packet filtering
	// http://en.wikipedia.org/wiki/Transmission_Control_Protocol
	destPort := binary.BigEndian.Uint16(buf[2:4])
	srcPort := binary.BigEndian.Uint16(buf[0:2])

	// Because RAW_SOCKET can't be bound to port, we have to control it by ourself
	if destPort == t.port || (t.trackResponse && srcPort == t.port) {
		// Get the 'data offset' (size of the TCP header in 32-bit words)
		dataOffset := (buf[12] & 0xF0) >> 4

		// We need only packets with data inside
		// Check that the buffer is larger than the size of the TCP header
		if len(buf) > int(dataOffset*4) {
			// We should create new buffer because go slices is pointers. So buffer data shoud be immutable.
			return true
		}
	}

	return false
}

var bExpect100ContinueCheck = []byte("Expect: 100-continue")
var bPOST = []byte("POST")

// Trying to add packet to existing message or creating new message
//
// For TCP message unique id is Acknowledgment number (see tcp_packet.go)
func (t *Listener) processTCPPacket(packet *TCPPacket) {
	// Don't exit on panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("PANIC: pkg:", r, packet, string(debug.Stack()))
		}
	}()

	// log.Println("Processing packet:", packet.Ack, packet.Seq, packet.ID)

	var message *TCPMessage

	isIncoming := packet.DestPort == t.port

	// Seek for 100-expect chunks
	if parentAck, ok := t.seqWithData[packet.Seq]; ok {
		// In case if non-first data chunks comes first
		for _, m := range t.messages {
			if m.Ack == packet.Ack && bytes.Equal(m.packets[0].Addr, packet.Addr) {
				t.deleteMessage(m)

				if m.AssocMessage != nil {
					m.AssocMessage.AssocMessage = nil
				}

				for _, pkt := range m.packets {
					// log.Println("Updating ack", parentAck, pkt.Ack)
					pkt.UpdateAck(parentAck)
					// Re-queue this packets
					t.processTCPPacket(pkt)
				}
			}
		}

		t.ackAliases[packet.Ack] = parentAck
		packet.UpdateAck(parentAck)
	}

	if alias, ok := t.ackAliases[packet.Ack]; ok {
		packet.UpdateAck(alias)
	}

	var responseRequest *TCPMessage

	if !isIncoming {
		responseRequest, _ = t.respAliases[packet.Ack]
	}

	message, ok := t.messages[packet.ID]

	if !ok {
		message = NewTCPMessage(packet.Seq, packet.Ack, isIncoming)
		t.messages[packet.ID] = message

		if !isIncoming {
			if responseRequest != nil {
				message.AssocMessage = responseRequest
				responseRequest.AssocMessage = message
			} else {
				t.respWithoutReq[packet.Ack] = packet.ID
			}
		}
	}

	// Adding packet to message
	message.AddPacket(packet)

	// Handling Expect: 100-continue requests
	if len(packet.Data) > 4 && bytes.Equal(packet.Data[0:4], bPOST) {
		// reading last 20 bytes (not counting CRLF): last header value (if no body presented)
		if bytes.Equal(packet.Data[len(packet.Data)-24:len(packet.Data)-4], bExpect100ContinueCheck) {
			seq := packet.Seq + uint32(len(packet.Data))
			t.seqWithData[seq] = packet.Ack
			message.DataSeq = seq

			// In case if sequence packet came first
			for _, m := range t.messages {
				if m.Seq == seq {
					t.deleteMessage(m)
					if m.AssocMessage != nil {
						message.AssocMessage = m.AssocMessage
					}
					// log.Println("2: Adding ack alias:", m.Ack, packet.Ack)
					t.ackAliases[m.Ack] = packet.Ack

					for _, pkt := range m.packets {
						pkt.UpdateAck(packet.Ack)
						message.AddPacket(pkt)
					}
				}
			}

			// Removing `Expect: 100-continue` header
			packet.Data = append(packet.Data[:len(packet.Data)-24], packet.Data[len(packet.Data)-2:]...)

			// log.Println(string(packet.Data))
		}
	}

	// log.Println("Received message:", string(message.Bytes()), message.ID(), t.messages)

	if isIncoming {
		// If message have multiple packets, delete previous alias
		if len(message.packets) > 1 {
			delete(t.respAliases, message.ResponseAck)
		}

		message.UpdateResponseAck()
		t.respAliases[message.ResponseAck] = message
	}

	// If message contains only single packet immediately dispatch it
	if message.IsFinished() {
		if isIncoming {
			// log.Println("I'm finished", string(message.Bytes()), message.ResponseID, t.messages)
			if t.trackResponse {
				if resp, ok := t.messages[message.ResponseID]; ok {
					t.dispatchMessage(message)
					if resp.IsFinished() {
						t.dispatchMessage(resp)
					}
				}
			} else {
				t.dispatchMessage(message)
			}
		} else {
			if message.AssocMessage == nil {
				return
			}

			if req, ok := t.messages[message.AssocMessage.ID()]; ok {
				if req.IsFinished() {
					t.dispatchMessage(req)
					t.dispatchMessage(message)
				}
			}
		}
	}
}

func (t *Listener) IsReady() bool {
	select {
	case <-t.readyCh:
		return true
	case <-time.After(5 * time.Second):
		return false
	}
}

// Receive TCP messages from the listener channel
func (t *Listener) Receiver() chan *TCPMessage {
	return t.messagesChan
}

func (t *Listener) Close() {
	for _, h := range t.connHandles {
		h.Close()
	}

	return
}
