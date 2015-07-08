package main

import (
	"bufio"
	"log"
	"net"
	"encoding/hex"
	"fmt"
	"os"
)

// Can be tested using nc tool:
//    echo "asdad" | nc 127.0.0.1 27017
//
type TCPInput struct {
	data     chan []byte
	address  string
	listener net.Listener
}

func NewTCPInput(address string) (i *TCPInput) {
	i = new(TCPInput)
	i.data = make(chan []byte)
	i.address = address

	i.listen(address)

	return
}

func (i *TCPInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	return len(buf), nil
}

func (i *TCPInput) listen(address string) {
	listener, err := net.Listen("tcp", address)
	i.listener = listener

	if err != nil {
		log.Fatal("Can't start:", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()

			if err != nil {
				log.Println("Error while Accept()", err)
				continue
			}

			go i.handleConnection(conn)
		}
	}()
}

func (i *TCPInput) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		encodedPayload := scanner.Bytes()
		// Hex encoding always 2x number of bytes
		decoded := make([]byte, len(encodedPayload)/2)
		hex.Decode(decoded, encodedPayload)
		i.data <- decoded
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Unexpected error in input tcp connection:", err)
	}
}

func (i *TCPInput) String() string {
	return "TCP input: " + i.address
}
