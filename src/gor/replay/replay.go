package replay

import (
    "bytes"
    "encoding/gob"
    "flag"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "strconv"
)

const (
    bufSize = 1024 * 10
)

type ReplaySettings struct {
    port int
    host string

    limit int

    address string
}

var settings ReplaySettings = ReplaySettings{}

type HttpRequest struct {
    Tag     string
    Method  string
    Url     string
    Headers map[string]string
}

type Response struct {
    req  *HttpRequest
    resp *http.Response
    err  error
}

func sendRequest(request *HttpRequest, responses chan *Response) {
    var req *http.Request

    client := &http.Client{}

    req, err := http.NewRequest("GET", settings.address+request.Url, nil)

    if err != nil {
        responses <- &Response{err: err}
        return
    }

    for key, value := range request.Headers {
        req.Header.Add(key, value)
    }

    resp, err := client.Do(req)

    responses <- &Response{request, resp, err}
}

func handleRequests(requests chan *HttpRequest) {
    stat := &RequestStat{}
    stat.reset()

    responses := make(chan *Response)

    for {
        select {
        case req := <-requests:
            go sendRequest(req, responses)

        case resp := <-responses:
            stat.inc(resp)

            if resp.err != nil {
                log.Println("Request err:", resp.err)
            } else {
                log.Println("Request ok:", resp.req.Url)
            }
        }
    }
}

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

    serverAddress := settings.host + ":" + strconv.Itoa(settings.port)

    addr, err := net.ResolveUDPAddr("udp", serverAddress)
    if err != nil {
        log.Fatal("Can't start:", err)
    }

    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        log.Fatal("Can't start:", err)
    }

    defer conn.Close()

    if settings.address == "" {

    }

    fmt.Println("Starting replay server at:", serverAddress)
    fmt.Println("Forwarding incoming requests to:", settings.address)
    fmt.Println("Limiting concurrent request count to:", settings.limit)

    requests := make(chan *HttpRequest)

    go handleRequests(requests)

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
                requests <- request
            }
        }
    }

}

func init() {
    if len(os.Args) > 1 && os.Args[1] != "replay" {
        return
    }

    const (
        defaultPort = 28020
        defaultHost = "0.0.0.0"

        defaultLimit   = 10
        defaultAddress = "http://localhost:8080"
    )

    flag.IntVar(&settings.port, "p", defaultPort, "specify port number")

    flag.StringVar(&settings.host, "ip", defaultHost, "ip addresses to listen on")

    flag.IntVar(&settings.limit, "l", defaultLimit, "limit number for requests per second. It will start dropping packets.")

    flag.StringVar(&settings.address, "h", defaultAddress, "http address to forward traffic ")
}
