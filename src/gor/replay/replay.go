package replay

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

var settings ReplaySettings = ReplaySettings{}

const bufSize = 1024 * 10

func decodeRequest(enc []byte) (request *HttpRequest, err error) {
	var buf bytes.Buffer
	buf.Write(enc)

	request = &HttpRequest{}

	encoder := gob.NewDecoder(&buf)
	err = encoder.Decode(request)

	return
}

func Run() {
	var buf [bufSize]byte

	addr, err := net.ResolveUDPAddr("udp", settings.Address())
	if err != nil {
		log.Fatal("Can't start:", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	fmt.Println("Starting replay server at:", settings.Address())

	if err != nil {
		log.Fatal("Can't start:", err)
	}

	defer conn.Close()

	for _, host := range settings.ForwardedHosts() {
		fmt.Println("Forwarding requests to:", host.Url, "limit:", host.Limit)
	}

	requestFactory := NewRequestFactory()

	for {
		rlen, _, err := conn.ReadFromUDP(buf[0:])

		if err != nil {
			continue
		}

		if rlen > 0 {
			if rlen > bufSize {
				log.Fatal("Too large udp packet", bufSize)
			}

			request, err := decodeRequest(buf[0:rlen])

			if err != nil {
				log.Println("Decode error:", err)
			} else {
				requestFactory.Add(request)
			}
		}
	}

}

func init() {
	if len(os.Args) < 2 || os.Args[1] != "replay" {
		return
	}

	const (
		defaultPort = 28020
		defaultHost = "0.0.0.0"

		defaultAddress = "http://localhost:8080"
	)

	flag.IntVar(&settings.port, "p", defaultPort, "specify port number")

	flag.StringVar(&settings.host, "ip", defaultHost, "ip addresses to listen on")

	flag.StringVar(&settings.address, "f", defaultAddress, "http address to forward traffic.\n\tYou can limit requests per second by adding `|num` after address.\n\tIf you have multiple addresses with different limits. For example: http://staging.example.com|100,http://dev.example.com|10")
}
