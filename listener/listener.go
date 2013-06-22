// Listener capture TCP traffic using RAW SOCKETS.
// Note: it requires sudo or root access.
//
// Rigt now it suport only HTTP
package listener

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// Enable debug logging only if "--verbose" flag passed
func Debug(v ...interface{}) {
	if Settings.verbose {
		log.Println(v...)
	}
}

func ReplayServer() net.Conn {
	// Connection to reaplay server
	conn, err := net.Dial("tcp", Settings.ReplayServer())

	if err != nil {
		log.Println("Connection error ", err, Settings.ReplayServer())
		log.Println("Reconnecting to replay server in 10 seconds")

		time.Sleep(10 * time.Second)
		return ReplayServer()
	}

	log.Println("Connected to replay server:", Settings.ReplayServer())

	return conn
}

// Because its sub-program, Run acts as `main`
func Run() {
	if os.Getuid() != 0 {
		fmt.Println("Please start the listener as root or sudo!")
		fmt.Println("This is required since listener sniff traffic on given port.")
		os.Exit(1)
	}

	fmt.Println("Listening for HTTP traffic on", Settings.Address())
	fmt.Println("Forwarding requests to replay server:", Settings.ReplayServer())

	// Sniffing traffic from given address
	listener := RAWTCPListen(Settings.address, Settings.port)

	for {
		// Receiving TCPMessage object
		m := listener.Receive()
		conn := ReplayServer()

		// For debugging purpose
		// Usually request parsing happens in replay part
		if Settings.verbose {
			buf := bytes.NewBuffer(m.Bytes())
			reader := bufio.NewReader(buf)

			request, err := http.ReadRequest(reader)

			if err != nil {
				Debug("Error while parsing request:", err, string(m.Bytes()))
			} else {
				request.ParseMultipartForm(32 << 20)
				Debug("Forwarding request:", request)
			}
		}

		_, err := conn.Write(m.Bytes())

		if err != nil {
			log.Println("Error while sending requests", err)
		}

		conn.Close()
	}
}
