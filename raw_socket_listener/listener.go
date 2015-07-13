package raw_socket

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"strconv"
)

// Capture traffic from socket using RAW_SOCKET's
// http://en.wikipedia.org/wiki/Raw_socket
//
// RAW_SOCKET allow you listen for traffic on any port (e.g. sniffing) because they operate on IP level.
// Ports is TCP feature, same as flow control, reliable transmission and etc.
// Since we can't use default TCP libraries RAWTCPLitener implements own TCP layer
// TCP packets is parsed using tcp_packet.go, and flow control is managed by tcp_message.go
type Listener struct {
	messages map[string]*TCPMessage // buffer of TCPMessages waiting to be send

	// Expect: 100-continue request is send in 2 tcp messages
	// We store ACK aliases to merge this packets together
	ack_aliases   map[uint32]uint32
	seq_with_data map[uint32]uint32

	c_packets  chan *TCPPacket
	c_messages chan *TCPMessage // Messages ready to be send to client

	c_del_message chan *TCPMessage // Used for notifications about completed or expired messages

	addr string // IP to listen
	port int    // Port to listen
}

// RAWTCPListen creates a listener to capture traffic from RAW_SOCKET
func NewListener(addr string, port string) (rawListener *Listener) {
	rawListener = &Listener{}

	rawListener.c_packets = make(chan *TCPPacket, 10000)
	rawListener.c_messages = make(chan *TCPMessage, 10000)
	rawListener.c_del_message = make(chan *TCPMessage, 10000)

	rawListener.messages = make(map[string]*TCPMessage)
	rawListener.ack_aliases = make(map[uint32]uint32)
	rawListener.seq_with_data = make(map[uint32]uint32)

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
		case message := <-t.c_del_message:
			t.c_messages <- message
			delete(t.ack_aliases, message.Ack)
			delete(t.messages, message.ID)

		// We need to use channels to process each packet to avoid data races
		case packet := <-t.c_packets:
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
		t.c_packets <- ParseTCPPacket(addr, buf)
	}
}

func (t *Listener) isIncomingDataPacket(buf []byte) bool {
	// To avoid full packet parsing every time, we manually parsing values needed for packet filtering
	// http://en.wikipedia.org/wiki/Transmission_Control_Protocol
	dest_port := binary.BigEndian.Uint16(buf[2:4])

	// Because RAW_SOCKET can't be bound to port, we have to control it by ourself
	if int(dest_port) == t.port {
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

	parent_message_ack, parent_ok := t.seq_with_data[packet.Seq]
	if parent_ok {
		t.ack_aliases[packet.Ack] = parent_message_ack
		delete(t.seq_with_data, packet.Seq)
	}

	ack_alias, alias_ok := t.ack_aliases[packet.Ack]
	if alias_ok {
		packet.Ack = ack_alias
	}

	m_id := packet.Addr.String() + strconv.Itoa(int(packet.SrcPort)) + strconv.Itoa(int(packet.Ack))
	message, ok := t.messages[m_id]

	if !ok {
		// We sending c_del_message channel, so message object can communicate with Listener and notify it if message completed
		message = NewTCPMessage(m_id, t.c_del_message, packet.Ack)
		t.messages[m_id] = message
	}

	if bytes.Equal(packet.Data[0:4], bPOST) {
		if bytes.Equal(packet.Data[len(packet.Data)-24:len(packet.Data)-4], bExpect100ContinueCheck) {
			t.seq_with_data[packet.Seq+uint32(len(packet.Data))] = packet.Ack

			// Removing `Expect: 100-continue` header
			packet.Data = append(packet.Data[:len(packet.Data)-24], packet.Data[len(packet.Data)-2:]...)
		}
	}

	// Adding packet to message
	message.c_packets <- packet
}

// Receive TCP messages from the listener channel
func (t *Listener) Receive() *TCPMessage {
	return <-t.c_messages
}
