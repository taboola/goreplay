package main

import (
	"crypto/rand"
	"encoding/base64"
)

type TestInput struct {
	data chan []byte
}

func NewTestInput() (i *TestInput) {
	i = new(TestInput)
	i.data = make(chan []byte, 100)

	return
}

func (i *TestInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	return len(buf), nil
}

func (i *TestInput) EmitGET() {
	i.data <- []byte("GET / HTTP/1.1\r\n\r\n")
}

func (i *TestInput) EmitPOST() {
	i.data <- []byte("POST /pub/WWW/ HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")
}

func (i *TestInput) EmitChunkedPOST() {
	i.data <- []byte("POST /pub/WWW/ HTTP/1.1\r\nHost: www.w3.org\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nWiki\r\n5\r\npedia\r\ne\r\n in\r\n\r\nchunks.\r\n0\r\n\r\n")
}

func (i *TestInput) EmitLargePOST() {
	size := 5 * 1024 * 1024 // 5 MB
	rb := make([]byte, size)
	rand.Read(rb)

	rs := base64.URLEncoding.EncodeToString(rb)

	i.data <- []byte("POST / HTTP/1.1\nHost: www.w3.org\nContent-Length:5242880\r\n\r\n" + rs)
}

func (i *TestInput) EmitOPTIONS() {
	i.data <- []byte("OPTIONS / HTTP/1.1\nHost: www.w3.org\r\n\r\n")
}

func (i *TestInput) String() string {
	return "Test Input"
}
