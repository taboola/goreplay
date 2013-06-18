package listener

import (
	"encoding/binary"
	"log"
	"net"
)

type RAWTCPListener struct {
	messages []*TCPMessage // buffer of TCPMessages waiting to be send

	c_packets  chan *TCPPacket
	c_messages chan *TCPMessage

	c_add_message chan *TCPMessage
	c_del_message chan *TCPMessage

	addr string
	port int
}

func RAWTCPListen(addr string, port int) (listener *RAWTCPListener) {
	listener = &RAWTCPListener{}

	listener.c_packets = make(chan *TCPPacket)
	listener.c_messages = make(chan *TCPMessage)
	listener.c_add_message = make(chan *TCPMessage)
	listener.c_del_message = make(chan *TCPMessage)

	listener.addr = addr
	listener.port = port

	go listener.listen()
	go listener.readTCPPackets()

	return
}

func (t *RAWTCPListener) listen() {
	for {
		select {
		case message := <-t.c_del_message:
			t.deleteMessage(message)
			Debug("Deleted")
			t.c_messages <- message

		case packet := <-t.c_packets:
			t.processTCPPacket(packet)
			Debug("Processed")
		}
	}
}

func (t *RAWTCPListener) deleteMessage(message *TCPMessage) bool {
	var idx int = -1

	for i, m := range t.messages {
		if m.Ack == message.Ack {
			idx = i
			break
		}
	}

	if idx == -1 {
		return false
	}

	copy(t.messages[idx:], t.messages[idx+1:])
	t.messages[len(t.messages)-1] = nil // or the zero value of T
	t.messages = t.messages[:len(t.messages)-1]

	return true
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
				flags := binary.BigEndian.Uint16(buf[12:14]) & 0x1FF
				f_psh := (flags & TCP_PSH) != 0

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
	var message *TCPMessage

	for _, msg := range t.messages {
		if msg.Ack == packet.Ack {
			message = msg
			break
		}
	}

	if message == nil {
		message = NewTCPMessage(packet.Ack, t.c_del_message)
		Debug("Adding message")

		t.messages = append(t.messages, message)
	}

	message.c_packets <- packet
}

func (t *RAWTCPListener) Receive() *TCPMessage {
	return <-t.c_messages
}
