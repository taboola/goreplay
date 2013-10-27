package gor

import (
	"log"
	"net"
)

type TCPOutput struct {
	address string
}

func NewTCPOutput(address string) (o *TCPOutput) {
	o = new(TCPOutput)
	o.address = address

	return
}

func (o *TCPOutput) Write(data []byte) (n int, err error) {
	conn, err := o.connect(o.address)
	defer conn.Close()

	n, err = conn.Write(data)

	return
}

func (o *TCPOutput) connect(address string) (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", address)

	if err != nil {
		log.Println("Connection error ", err, o.address)
	}

	return
}

func (o *TCPOutput) String() string {
	return "TCP output: " + o.address
}
