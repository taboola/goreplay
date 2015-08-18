package main

import (
	"time"
)

// DummyInput used for debugging. It generate 1 "GET /"" request per second.
type DummyInput struct {
	data chan []byte
}

// NewDummyInput constructor for DummyInput
func NewDummyInput(options string) (di *DummyInput) {
	di = new(DummyInput)
	di.data = make(chan []byte)

	go di.emit()

	return
}

func (i *DummyInput) Read(data []byte) (int, error) {
	buf := <-i.data

	header := payloadHeader(RequestPayload, uuid(), time.Now().UnixNano())
	copy(data[0:len(header)], header)
	copy(data[len(header):], buf)

	return len(buf) + len(header), nil
}

func (i *DummyInput) emit() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			i.data <- []byte("POST /pub/WWW/å HTTP/1.1\nHost: www.w3.org\r\n\r\na=1&b=2")
		}
	}
}

func (i *DummyInput) String() string {
	return "Dummy Input"
}
