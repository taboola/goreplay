package gor

import (
	raw "github.com/buger/gor/raw_socket_listener"
	"log"
	"net"
)

type RAWInput struct {
	data chan []byte
}

func NewRAWInput(address string) (i *RAWInput) {
	i = new(RAWInput)
	i.data = make(chan []byte)

	go i.listen(address)

	return
}

func (i *RAWInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	log.Println("Sending message", buf)

	return len(buf), nil
}

func (i *RAWInput) listen(address string) {
	host, port, err := net.SplitHostPort(address)

	if err != nil {
		log.Fatal("input-raw: error while parsing address", err)
	}

	listener := raw.NewListener(host, port)

	for {
		// Receiving TCPMessage object
		m := listener.Receive()

		i.data <- m.Bytes()
	}
}
