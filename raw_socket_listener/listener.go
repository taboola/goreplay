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
	"log"
	"net"
	"strconv"
)

// Listener handle traffic capture
type Listener struct {
	// buffer of TCPMessages waiting to be send
	messages map[string]*TCPMessage

	// Expect: 100-continue request is send in 2 tcp messages
	// We store ACK aliases to merge this packets together
	ackAliases map[uint32]uint32
	// To get ACK of second message we need to compute its Seq and wait for them message
	seqWithData map[uint32]uint32

	// Messages ready to be send to client
	packetsChan chan *TCPPacket

	// Messages ready to be send to client
	messagesChan chan *TCPMessage

	// Used for notifications about completed or expired messages
	messageDelChan chan *TCPMessage

	addr string // IP to listen
	port int    // Port to listen
}

// NewListener creates and initializes new Listener object
func NewListener(addr string, port string) (rawListener *Listener) {
	rawListener = &Listener{}

	rawListener.packetsChan = make(chan *TCPPacket, 10000)
	rawListener.messagesChan = make(chan *TCPMessage, 10000)
	rawListener.messageDelChan = make(chan *TCPMessage, 10000)

	rawListener.messages = make(map[string]*TCPMessage)
	rawListener.ackAliases = make(map[uint32]uint32)
	rawListener.seqWithData = make(map[uint32]uint32)

	rawListener.addr = addr
	rawListener.port, _ = strconv.Atoi(port)

	go rawListener.listen()
	go rawListener.readRAWSocket()

	return
}

func (t *Listener) listen() {
	for {
		select {
		// If message ready for deletion it means that its also complete or expired by timeout
		case message := <-t.messageDelChan:
			t.messagesChan <- message
			delete(t.ackAliases, message.Ack)
			delete(t.messages, message.ID)

		// We need to use channels to process each packet to avoid data races
		case packet := <-t.packetsChan:
			t.processTCPPacket(packet)
		}
	}
}
func (t *Listener) readRAWSocket() {
	conn, e := net.ListenPacket("ip4:tcp", t.addr)

	if e != nil {
		log.Fatal(e)
	}

	defer conn.Close()

	for {
		buf := make([]byte, 64*1024) // 64kb
		// Note: ReadFrom receive messages without IP header
		n, addr, err := conn.ReadFrom(buf)

		if err != nil {
			log.Println("Error:", err)
			continue
		}

		if n > 0 {
			go t.parsePacket(addr, buf[:n])
		}
	}
}

func (t *Listener) parsePacket(addr net.Addr, buf []byte) {
	if t.isIncomingDataPacket(buf) {
		t.packetsChan <- ParseTCPPacket(addr, buf)
	}
}

func (t *Listener) isIncomingDataPacket(buf []byte) bool {
	// To avoid full packet parsing every time, we manually parsing values needed for packet filtering
	// http://en.wikipedia.org/wiki/Transmission_Control_Protocol
	destPort := binary.BigEndian.Uint16(buf[2:4])

	// Because RAW_SOCKET can't be bound to port, we have to control it by ourself
	if int(destPort) == t.port {
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
	defer func() { recover() }()

	var message *TCPMessage

	if parentAck, ok := t.seqWithData[packet.Seq]; ok {
		t.ackAliases[packet.Ack] = parentAck
		delete(t.seqWithData, packet.Seq)
	}

	if alias, ok := t.ackAliases[packet.Ack]; ok {
		packet.Ack = alias
	}

	mID := packet.Addr.String() + strconv.Itoa(int(packet.SrcPort)) + strconv.Itoa(int(packet.Ack))
	message, ok := t.messages[mID]

	if !ok {
		// We sending messageDelChan channel, so message object can communicate with Listener and notify it if message completed
		message = NewTCPMessage(mID, t.messageDelChan, packet.Ack)
		t.messages[mID] = message
	}

	if bytes.Equal(packet.Data[0:4], bPOST) {
		if bytes.Equal(packet.Data[len(packet.Data)-24:len(packet.Data)-4], bExpect100ContinueCheck) {
			t.seqWithData[packet.Seq+uint32(len(packet.Data))] = packet.Ack

			// Removing `Expect: 100-continue` header
			packet.Data = append(packet.Data[:len(packet.Data)-24], packet.Data[len(packet.Data)-2:]...)
		}
	}

	// Adding packet to message
	message.packetsChan <- packet
}

// Receive TCP messages from the listener channel
func (t *Listener) Receive() *TCPMessage {
	return <-t.messagesChan
}
