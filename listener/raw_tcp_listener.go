package listener

import (
	"encoding/binary"
	"log"
	"net"
	"time"
)

type RAWTCPListener struct {
	messages map[uint32]*TCPMessage // buffer of TCPMessages waiting to be send

	c_packets  chan *TCPPacket
	c_messages chan *TCPMessage

	addr string
	port int
}

func RAWTCPListen(addr string, port int) (listener *RAWTCPListener) {
	listener = &RAWTCPListener{}
	listener.messages = make(map[uint32]*TCPMessage)

	listener.c_packets = make(chan *TCPPacket)
	listener.c_messages = make(chan *TCPMessage)

	listener.addr = addr
	listener.port = port

	go listener.listen()
	go listener.readTCPPackets()

	return
}

func (t *RAWTCPListener) listen() {

	for {
		var messages chan *TCPMessage
		var message *TCPMessage

		for _, msg := range t.messages {
			if msg.Complete() {
				messages = t.c_messages
				message = msg
				break
			}
		}

		select {
		case messages <- message:
			delete(t.messages, message.ask)
		case packet := <-t.c_packets:
			t.processTCPPacket(packet)

		// Ensure that this will be run at least each 200 ms, to ensure that all messages will be send
		// Without it last message may not be send (it will be send only on next TCP packets)
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (t *RAWTCPListener) readTCPPackets() {
	conn, e := net.ListenPacket("ip4:tcp", t.addr)
	defer conn.Close()

	if e != nil {
		log.Fatal(e)
	}

	buf := make([]byte, 1500*2)

	for {
		n, _, err := conn.ReadFrom(buf)

		if err != nil {
			Debug("Error:", err)
		}

		if n > 0 {
			// http://en.wikipedia.org/wiki/Transmission_Control_Protocol
			dest_port := binary.BigEndian.Uint16(buf[2:4])

			if int(dest_port) == t.port {
				// Check TCPPacket code for more description
				doff := binary.BigEndian.Uint16(buf[12:14])
				f_psh := (doff & 8) != 0

				// We need only packets with data inside
				// TCP PSH flag indicate that client should push data to buffer
				if f_psh {
					new_buf := make([]byte, n)
					copy(new_buf, buf[:n])

					packet := NewTCPPacket(new_buf)

					t.c_packets <- packet
				}
			}
		}
	}
}

//
func (t *RAWTCPListener) processTCPPacket(packet *TCPPacket) {
	// We interested only in packets that contain some data
	if !(packet.f_ask && packet.f_psh) {
		return
	}

	ask := packet.asknowledgement

	if _, ok := t.messages[ask]; !ok {
		t.messages[ask] = NewTCPMessage(ask)
	}

	t.messages[ask].AddPacket(packet)
}

func (t *RAWTCPListener) Receive() *TCPMessage {
	return <-t.c_messages
}
