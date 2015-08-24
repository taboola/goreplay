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

	copy(data, buf)

	return len(buf), nil
}

func (i *DummyInput) emit() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			uuid := uuid()
			reqh := payloadHeader(RequestPayload, uuid, time.Now().UnixNano())
			i.data <- append(reqh, []byte("POST /pub/WWW/å HTTP/1.1\nHost: www.w3.org\r\nContent-Length: 7\r\n\r\na=1&b=2")...)

			resh := payloadHeader(ResponsePayload, uuid, 1)
			i.data <- append(resh, []byte("HTTP/1.1 200 OK\r\n\r\n")...)
		}
	}
}

func (i *DummyInput) String() string {
	return "Dummy Input"
}
