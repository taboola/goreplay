package main

import (
	raw "github.com/buger/gor/raw_socket_listener"
	"log"
	"net"
	"strings"
	"time"
)

// RAWInput used for intercepting traffic for given address
type RAWInput struct {
	data    chan []byte
	address string
	expire  time.Duration
}

// NewRAWInput constructor for RAWInput. Accepts address with port as argument.
func NewRAWInput(address string, expire time.Duration) (i *RAWInput) {
	i = new(RAWInput)
	i.data = make(chan []byte)
	i.address = address
	i.expire = expire

	go i.listen(address)

	return
}

func (i *RAWInput) Read(data []byte) (int, error) {
	buf := <-i.data

	if len(Settings.middleware) > 0 {
		header := []byte("1\n")
		copy(data[0:len(header)], header)
		copy(data[len(header):], buf)
	} else {
		copy(data, buf)
	}

	return len(buf), nil
}

func (i *RAWInput) listen(address string) {
	address = strings.Replace(address, "[::]", "127.0.0.1", -1)

	Debug("Listening for traffic on: " + address)

	host, port, err := net.SplitHostPort(address)

	if err != nil {
		log.Fatal("input-raw: error while parsing address", err)
	}

	listener := raw.NewListener(host, port, i.expire)

	for {
		// Receiving TCPMessage object
		m := listener.Receive()

		i.data <- m.Bytes()
	}
}

func (i *RAWInput) String() string {
	return "RAW Socket input: " + i.address
}
