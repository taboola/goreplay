// Listener capture TCP traffic right from given port using `tcpdump` utility.
// Note: it requires sudo or root access.
//
// Rigt now it suport only HTTP, and only GET requests.
package listener

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type ListenerSettings struct {
	networkInterface string
	port             int

	replayAddress string
}

var settings ListenerSettings = ListenerSettings{}

type HttpRequest struct {
	Tag     string            // Not used yet
	Method  string            // Right now only 'GET'
	Url     string            // Request URL
	Headers map[string]string // Request Headers
}

// Parse `tcpdump` output to find HTTP GET requests
// When HttpRequest found it get send to `requests` channel
func ParseRequest(pipe io.ReadCloser, requests chan *HttpRequest) {
	request_re := regexp.MustCompile("(GET) (/.*) HTTP/1.1")
	headers_re := regexp.MustCompile("([^ ]*): (.*)")

	reader := bufio.NewScanner(pipe)

	var request *HttpRequest

	var requestStarted = false

	for reader.Scan() {
		line := reader.Text()

		// HTTP/1.1 match finds both requests and response
		// Index is used instead of Regexp just for speed
		if strings.Index(line, "HTTP/1.1") != -1 {
			// Allow only requests
			match := request_re.FindAllString(line, -1)

			if len(match) > 0 {
				info := strings.Split(match[0], " ")

				request = &HttpRequest{
					Method:  info[0],
					Url:     info[1],
					Headers: make(map[string]string),
				}

				requestStarted = true
			}
		}

		if requestStarted {
			// We assume that empty line is end of request info
			// This is true only for GET requests
			if line == "" {
				requests <- request
				requestStarted = false
			} else {
				// All headers comes in this format:
				//
				//     User-Agent: Mozilla
				//     Content-Type: text/html
				//
				match := headers_re.FindAllString(line, -1)

				if len(match) > 0 {
					header := strings.Split(match[0], ": ")

					request.Headers[header[0]] = header[1]
				}
			}
		}
	}
}

// Sends request to replay server via UDP
// Before sending it encode request object using standard gob encoder
func SendRequest(requests chan *HttpRequest) {
	serverAddr, err := net.ResolveUDPAddr("udp4", settings.replayAddress)
	conn, err := net.DialUDP("udp", nil, serverAddr)

	defer conn.Close()

	if err != nil {
		log.Fatal("Connection error", err)
	}

	for {
		select {
		case request := <-requests:
			fmt.Println("Request:", request.Url)

			msg := bytes.Buffer{}

			enc := gob.NewEncoder(&msg)
			err := enc.Encode(request)

			conn.Write(msg.Bytes())

			if err != nil {
				log.Println("encode error:", err)
			}
		}
	}
}

// Because its sub-program, Run acts as `main`
func Run() {
	if os.Getuid() != 0 {
		fmt.Println("Please start the listener as root or sudo!")
		fmt.Println("This is required since listener sniff traffic on given port.")
		os.Exit(1)
	}

	if !strings.Contains(settings.replayAddress, ":") {
		settings.replayAddress = settings.replayAddress + ":28020"
	}

	// TODO: use RAW_SOCKETS instead of tcpdump
	cmd := exec.Command("tcpdump", "-vv", "-A", "-i", settings.networkInterface, "port "+strconv.Itoa(settings.port))

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	requests := make(chan *HttpRequest)

	go ParseRequest(stdout, requests)
	go SendRequest(requests)

	if err := cmd.Wait(); err != nil {
		flag.Usage()
	}
}

func init() {
	if len(os.Args) < 2 || os.Args[1] != "listen" {
		return
	}

	const (
		defaultPort             = 80
		defaultNetworkInterface = "any"

		defaultReplayAddress = "localhost:28020"
	)

	flag.IntVar(&settings.port, "p", defaultPort, "Specify the http server port whose traffic you want to capture")

	flag.StringVar(&settings.networkInterface, "i", defaultNetworkInterface, "By default it try to listen on all network interfaces.To get list of interfaces run `ifconfig`")

	flag.StringVar(&settings.replayAddress, "r", defaultReplayAddress, "Address of replay server.")
}
