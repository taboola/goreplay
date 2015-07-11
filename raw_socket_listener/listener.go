package raw_socket

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"
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
			log.Println("Sending message, len:", len(message.packets))
			t.c_messages <- message
			delete(t.ack_aliases, message.Ack)
			delete(t.messages, message.ID)

		// We need to use channels to process each packet to avoid data races
		case packet := <-t.c_packets:
			t.processTCPPacket(packet)
		}
	}
}

// Taken from http://golang.org/src/net/sock_cloexec.go?h=sysSocket#L16
func sysSocket(family, sotype, proto int) (int, error) {
	s, err := syscall.Socket(family, sotype|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, proto)
	// On Linux the SOCK_NONBLOCK and SOCK_CLOEXEC flags were
	// introduced in 2.6.27 kernel and on FreeBSD both flags were
	// introduced in 10 kernel. If we get an EINVAL error on Linux
	// or EPROTONOSUPPORT error on FreeBSD, fall back to using
	// socket without them.
	if err == nil || (err != syscall.EPROTONOSUPPORT && err != syscall.EINVAL) {
		return s, err
	}

	// See ../syscall/exec_unix.go for description of ForkLock.
	syscall.ForkLock.RLock()
	s, err = syscall.Socket(family, sotype, proto)
	if err == nil {
		syscall.CloseOnExec(s)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return -1, err
	}
	if err = syscall.SetNonblock(s, true); err != nil {
		syscall.Close(s)
		return -1, err
	}
	return s, nil
}

func ipToSockaddr(ip net.IP) (syscall.Sockaddr, error) {
	if len(ip) == 0 {
		ip = net.IPv4zero
	}
	if ip = ip.To4(); ip == nil {
		return nil, net.InvalidAddrError("non-IPv4 address")
	}

	sa := new(syscall.SockaddrInet4)
	for i := 0; i < net.IPv4len; i++ {
		sa.Addr[i] = ip[i]
	}
	sa.Port = 0
	return sa, nil
}

func FD_SET(p *syscall.FdSet, i int) {
	p.Bits[i/64] |= 1 << uint(i) % 64
}

func FD_ISSET(p *syscall.FdSet, i int) bool {
	return (p.Bits[i/64] & (1 << uint(i) % 64)) != 0
}

func FD_ZERO(p *syscall.FdSet) {
	for i := range p.Bits {
		p.Bits[i] = 0
	}
}

func (t *Listener) readRAWSocket() {
	var err error
	var n int
	var sa syscall.Sockaddr

	addr, _ := net.ResolveIPAddr("ip4", t.addr)
	sa, _ = ipToSockaddr(addr.IP)
	fd, e := sysSocket(syscall.AF_INET, syscall.SOCK_RAW|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, syscall.IPPROTO_TCP)
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)

	syscall.SetNonblock(fd, true)

	if e != nil {
		log.Fatal(e)
	}

	if err := syscall.Bind(fd, sa); err != nil {
		log.Fatal(os.NewSyscallError("bind", err))
	}

	defer syscall.Close(fd)

	rfds := &syscall.FdSet{}
	timeout := syscall.NsecToTimeval(time.Second.Nanoseconds())

	for {
		buf := make([]byte, 64*1024) // 64kb

		for {
			if _, err := syscall.Select(fd, rfds, nil, nil, &timeout); err != nil {
				log.Fatal("Error", e)
			}

			n, sa, err = syscall.Recvfrom(fd, buf, 0)

			if err != nil {
				if err == syscall.EAGAIN {
					n = 0
					continue
				}
			}

			break
		}

		if err != nil {
			log.Println("Error:", err)
			continue
		}

		if n > 0 {
			// Ip header size
			hsize := (int(buf[0]) & 0xf) * 4

			if n > hsize {
				go t.parsePacket(sa, buf[hsize:n])
			}
		}

	}
}

func (t *Listener) parsePacket(sa syscall.Sockaddr, buf []byte) {
	addr := &net.IPAddr{IP: sa.(*syscall.SockaddrInet4).Addr[0:]}

	if t.isIncomingDataPacket(buf) {
		log.Println("Received packet:", len(buf))
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

	m_id := packet.Addr.String() + strconv.Itoa(int(packet.Ack))
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
