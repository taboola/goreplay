package main

import (
	"time"
)

type DummyInput struct {
	data chan []byte
}

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
			i.data <- []byte("POST /pub/WWW/å HTTP/1.1\nHost: www.w3.org\r\n\r\na=1&b=2")
		}
	}
}

func (i *DummyInput) String() string {
	return "Dummy Input"
}
