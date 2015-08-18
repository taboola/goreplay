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
	data            chan *raw.TCPMessage
	address         string
	expire          time.Duration
}

// NewRAWInput constructor for RAWInput. Accepts address with port as argument.
func NewRAWInput(address string, expire time.Duration) (i *RAWInput) {
	i = new(RAWInput)
	i.data = make(chan *raw.TCPMessage)
	i.address = address
	i.expire = expire

	go i.listen(address)

	return
}

func (i *RAWInput) Read(data []byte) (int, error) {
	msg := <-i.data
	buf := msg.Bytes()

	var header []byte

	if msg.IsIncoming {
		header = payloadHeader(RequestPayload, msg.UUID(), msg.Start)
	} else {
		header = payloadHeader(ResponsePayload, msg.UUID(), msg.End-msg.RequestStart)
	}

	copy(data[0:len(header)], header)
	copy(data[len(header):], buf)

	return len(buf) + len(header), nil
}

func (i *RAWInput) listen(address string) {
	address = strings.Replace(address, "[::]", "127.0.0.1", -1)

	Debug("Listening for traffic on: " + address)

	host, port, err := net.SplitHostPort(address)

	if err != nil {
		log.Fatal("input-raw: error while parsing address", err)
	}

	listener := raw.NewListener(host, port, i.expire, true)

	for {
		// Receiving TCPMessage object
		m := listener.Receive()

		i.data <- m
	}
}

func (i *RAWInput) String() string {
	return "RAW Socket input: " + i.address
}
