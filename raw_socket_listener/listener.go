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
	"github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"io"
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

var _ = fmt.Println

// Listener handle traffic capture
type Listener struct {
	mu sync.Mutex
	// buffer of TCPMessages waiting to be send
	// ID -> TCPMessage
	messages map[string]*TCPMessage

	// Expect: 100-continue request is send in 2 tcp messages
	// We store ACK aliases to merge this packets together
	ackAliases map[uint32]uint32
	// To get ACK of second message we need to compute its Seq and wait for them message
	seqWithData map[uint32]uint32

	// Ack -> Req
	respAliases map[uint32]*request

	// Ack -> ID
	respWithoutReq map[uint32]string

	// Messages ready to be send to client
	packetsChan chan *TCPPacket

	// Messages ready to be send to client
	messagesChan chan *TCPMessage

	addr string // IP to listen
	port uint16 // Port to listen

	messageExpire time.Duration

	conn net.PacketConn
	quit chan bool
}

type request struct {
	id string
	start time.Time
	ack   uint32
}

// Available engines for intercepting traffic
const (
	EngineRawSocket = 1 << iota
	EnginePcap
)

// NewListener creates and initializes new Listener object
func NewListener(addr string, port string, engine int, expire time.Duration) (l *Listener) {
	l = &Listener{}

	l.packetsChan = make(chan *TCPPacket, 10000)
	l.messagesChan = make(chan *TCPMessage, 10000)
	l.quit = make(chan bool)

	l.messages = make(map[string]*TCPMessage)
	l.ackAliases = make(map[uint32]uint32)
	l.seqWithData = make(map[uint32]uint32)
	l.respAliases = make(map[uint32]*request)
	l.respWithoutReq = make(map[uint32]string)

	l.addr = addr
	_port, _ := strconv.Atoi(port)
	l.port = uint16(_port)

	if expire.Nanoseconds() == 0 {
		expire = 2000 * time.Millisecond
	}

	l.messageExpire = expire

	go l.listen()
	go l.processPackets()

	// Special case for testing
	if l.port != 0 {
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

func (t *Listener) processPackets() {
	for {
		// We need to use channels to process each packet to avoid data races
		packet := <-t.packetsChan
		// log.Println(packet)
		t.mu.Lock()
		t.processTCPPacket(packet)
		t.mu.Unlock()
	}
}

func (t *Listener) listen() {
	gcTicker := time.Tick(t.messageExpire / 2)

	for {
		select {
		case <-t.quit:
			if t.conn != nil {
				t.conn.Close()
			}
			return
		case <-gcTicker:
			now := time.Now()
			// log.Println("GC")

			t.mu.Lock()
			// Dispatch requests before responses
			for _, message := range t.messages {
				if now.Sub(message.End) >= t.messageExpire {
					t.dispatchMessage(message)
				}
			}

			t.mu.Unlock()
		}
	}
}

func (t *Listener) dispatchMessage(message *TCPMessage) {
	// If already dispatched
	if _, ok := t.messages[message.ID]; !ok {
		return
	}

	delete(t.ackAliases, message.Ack)
	delete(t.messages, message.ID)

	// log.Println("Dispatching, message", message.Seq, message.Ack, string(message.Bytes()))

	if message.IsIncoming {
		// If there were response before request
		// log.Println("Looking for Response: ", t.respWithoutReq, message.ResponseAck)
		if respID, ok := t.respWithoutReq[message.ResponseAck]; ok {
			if resp, rok := t.messages[respID]; rok {
				if resp.RequestAck == 0 {
					// log.Println("FOUND RESPONSE")
					resp.RequestAck = message.Ack
					resp.RequestStart = message.Start

					if resp.IsFinished() {
						defer t.dispatchMessage(resp)
					}
				}
			}
		}
	} else {
		if message.RequestAck == 0 {
			if responseRequest, ok := t.respAliases[message.Ack]; ok {
				message.RequestStart = responseRequest.start
				message.RequestAck = responseRequest.ack
			}
		}

		delete(t.respAliases, message.Ack)
		delete(t.respWithoutReq, message.Ack)

		// Do not track responses which have no associated requests
		if message.RequestAck == 0 {
			// log.Println("Can't dispatch resp", message.Seq, message.Ack, string(message.Bytes()))
			return
		}
	}

	t.messagesChan <- message
}

// DeviceNotFoundError raised if user specified wrong ip
type DeviceNotFoundError struct {
	addr string
}

func (e *DeviceNotFoundError) Error() string {
	devices, _ := pcap.FindAllDevs()

	var msg string
	msg += "Devices with addr: " + e.addr + " not found. Available devices: \n"
	for _, device := range devices {
		msg += "Name: " + device.Name + "\n"
		msg += "Description: " + device.Description + "\n"
		msg += "Devices addresses: " + device.Description + "\n"
		for _, address := range device.Addresses {
			msg += "- IP address: " + address.IP.String() + "\n"
			msg += "- Subnet mask: " + address.Netmask.String() + "\n"
		}
	}

	return msg
}

func findPcapDevice(addr string) (*pcap.Interface, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range devices {
		if device.Name == "any" && addr == "" || addr == "0.0.0.0" {
			return &device, nil
		}

		for _, address := range device.Addresses {
			if address.IP.String() == addr {
				return &device, nil
			}
		}
	}

	return nil, &DeviceNotFoundError{addr}
}

func (t *Listener) readPcap() {
	device, err := findPcapDevice(t.addr)
	if err != nil {
		log.Fatal(err)
	}

	handle, err := pcap.OpenLive(device.Name, 65536, true, t.messageExpire)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter("tcp and port " + strconv.Itoa(int(t.port))); err != nil {
		log.Fatal(err)
	}

	source := gopacket.NewPacketSource(handle, handle.LinkType())
	source.Lazy = true
	source.NoCopy = true

	// log.Println(handle.Stats())

	for {
		packet, err := source.NextPacket()

		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}

		// Skip ethernet layer, 14 bytes
		data := packet.Data()[14:]
		ihl := uint8(data[0]) & 0x0F
		srcIP := data[12:16]
		data = data[ihl*4:]

		dataOffset := (data[12] & 0xF0) >> 4

		// We need only packets with data inside
		// Check that the buffer is larger than the size of the TCP header
		if len(data) > int(dataOffset*4) {
			newBuf := make([]byte, len(data))
			copy(newBuf, data)

			go func(newBuf []byte) {
				t.packetsChan <- ParseTCPPacket(net.IP(srcIP).String(), newBuf)
			}(newBuf)
		}
	}
}

func (t *Listener) readRAWSocket() {
	conn, e := net.ListenPacket("ip4:tcp", t.addr)
	t.conn = conn

	if e != nil {
		log.Fatal(e)
	}

	defer t.conn.Close()

	buf := make([]byte, 64*1024) // 64kb

	for {
		// Note: ReadFrom receive messages without IP header
		n, addr, err := t.conn.ReadFrom(buf)

		if err != nil {
			if strings.HasSuffix(err.Error(), "closed network connection") {
				return
			} else {
				continue
			}
		}

		if n > 0 {
			if t.isValidPacket(buf[:n]) {
				newBuf := make([]byte, n)
				copy(newBuf, buf[:n])

				go func(newBuf []byte) {
					t.packetsChan <- ParseTCPPacket(addr.String(), newBuf)
				}(newBuf)
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
	if destPort == t.port || srcPort == t.port {
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

	// log.Println("Processing packet:", packet.Ack, packet.Seq, string(packet.Data))

	var message *TCPMessage

	isIncoming := packet.DestPort == t.port

	// Seek for 100-expect chunks
	if parentAck, ok := t.seqWithData[packet.Seq]; ok {
		// log.Println("Found data package with Ack:", packet.Ack)
		// In case if non-first data chunks comes first
		for _id, m := range t.messages {
			// log.Println("Message ack:", m.Ack, m.packets[0].Addr, packet.Addr)
			if m.Ack == packet.Ack && m.packets[0].Addr == packet.Addr {
				delete(t.messages, _id)

				for _, pkt := range m.packets {
					pkt.Ack = parentAck
					// Re-queue this packets
					t.processTCPPacket(pkt)
				}
			}
		}

		delete(t.seqWithData, packet.Seq)
		t.ackAliases[packet.Ack] = parentAck
		packet.Ack = parentAck
	}

	if alias, ok := t.ackAliases[packet.Ack]; ok {
		packet.Ack = alias
	}

	var responseRequest *request

	if !isIncoming {
		responseRequest, _ = t.respAliases[packet.Ack]
	}

	mID := packet.Addr + strconv.Itoa(int(packet.DestPort)) + strconv.Itoa(int(packet.Ack))

	message, ok := t.messages[mID]

	if !ok {
		message = NewTCPMessage(mID, packet.Seq, packet.Ack, isIncoming)
		t.messages[mID] = message

		if !isIncoming {
			if responseRequest != nil {
				message.RequestStart = responseRequest.start
				message.RequestAck = responseRequest.ack
				message.RequestID = responseRequest.id
			} else {
				t.respWithoutReq[packet.Ack] = mID
			}
		}
	}

	// Handling Expect: 100-continue requests
	if len(packet.Data) > 4 && bytes.Equal(packet.Data[0:4], bPOST) {
		// reading last 20 bytes (not counting CRLF): last header value (if no body presented)
		if bytes.Equal(packet.Data[len(packet.Data)-24:len(packet.Data)-4], bExpect100ContinueCheck) {
			seq := packet.Seq + uint32(len(packet.Data))
			t.seqWithData[seq] = packet.Ack

			// In case if sequence packet came first
			// log.Println("Looking for sequences:", seq, t.messages)
			for _id, m := range t.messages {
				// log.Println("SeqSEQ", m.Seq, len(m.packets))
				if m.Seq == seq {
					t.ackAliases[m.Ack] = packet.Ack

					for _, pkt := range m.packets {
						message.AddPacket(pkt)
					}

					delete(t.messages, _id)
				}
			}

			// Removing `Expect: 100-continue` header
			packet.Data = append(packet.Data[:len(packet.Data)-24], packet.Data[len(packet.Data)-2:]...)

			// log.Println(string(packet.Data))
		}
	}

	// Adding packet to message
	message.AddPacket(packet)

	if isIncoming {
		// If message have multiple packets, delete previous alias
		if len(message.packets) > 1 {
			delete(t.respAliases, message.ResponseAck)
		}

		message.UpdateResponseAck()
		t.respAliases[message.ResponseAck] = &request{message.ID, message.Start, message.Ack}
	}

	// If message contains only single packet immediately dispatch it
	if message.IsFinished() {
		if isIncoming {
			if resp, ok := t.messages[message.ResponseID()]; ok {
				t.dispatchMessage(message)
				if resp.IsFinished() {
					t.dispatchMessage(resp)
				}
			}
		} else {
			if req, ok := t.messages[message.RequestID]; ok {
				if req.IsFinished() {
					t.dispatchMessage(req)
					t.dispatchMessage(message)
				}
			}
		}
	}
}

// Receive TCP messages from the listener channel
func (t *Listener) Receive() *TCPMessage {
	return <-t.messagesChan
}

func (t *Listener) Close() {
	close(t.quit)
	if t.conn != nil {
		t.conn.Close()
	}
	return
}
