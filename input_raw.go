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
	requests    chan []byte
	responses    chan []byte
	address string
	expire  time.Duration
	captureResponse bool
}

// NewRAWInput constructor for RAWInput. Accepts address with port as argument.
func NewRAWInput(address string, expire time.Duration, captureResponse bool) (i *RAWInput) {
	i = new(RAWInput)
	i.requests = make(chan []byte)
	i.responses = make(chan []byte)
	i.address = address
	i.expire = expire
	i.captureResponse = captureResponse

	go i.listen(address)

	return
}

func (i *RAWInput) Read(data []byte) (int, error) {
	select {
	case buf := <- i.requests:
		if i.captureResponse {
			header := []byte("1\n")
			copy(data[0:len(header)], header)
			copy(data[len(header):], buf)

			return len(buf) + len(header), nil
		} else {
			copy(data, buf)

			return len(buf), nil
		}
	case buf := <- i.responses:
		header := []byte("3\n")
		copy(data[0:len(header)], header)
		copy(data[len(header):], buf)

		return len(buf) + len(header), nil
	}
}

func (i *RAWInput) listen(address string) {
	address = strings.Replace(address, "[::]", "127.0.0.1", -1)

	Debug("Listening for traffic on: " + address)

	host, port, err := net.SplitHostPort(address)

	if err != nil {
		log.Fatal("input-raw: error while parsing address", err)
	}

	listener := raw.NewListener(host, port, i.expire, i.captureResponse)

	for {
		// Receiving TCPMessage object
		m := listener.Receive()

		i.requests <- m.RequestBytes()

		if i.captureResponse {
			i.responses <- m.ResponseBytes()
		}
	}
}

func (i *RAWInput) String() string {
	return "RAW Socket input: " + i.address
}
