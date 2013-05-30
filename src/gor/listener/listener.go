package listener

import (
    "fmt"
    "io"
    "log"
    "os/exec"
    //"time"
    "bufio"
    "bytes"
    "encoding/gob"
    //"errors"
    "flag"
    "net"
    "os"
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
    Tag     string
    Method  string
    Url     string
    Headers map[string]string
}

func readOutput(pipe io.ReadCloser, c chan *HttpRequest, err chan int) {
    re := regexp.MustCompile("(GET) (/.*) HTTP/1.1")
    headers_re := regexp.MustCompile("([^ ]*): (.*)")

    reader := bufio.NewScanner(pipe)

    var request *HttpRequest

    var requestStarted = false

    for reader.Scan() {
        line := reader.Text()

        if strings.Index(line, "HTTP/1.1") != -1 {
            match := re.FindAllString(line, -1)

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
            if line == "" {
                c <- request
                requestStarted = false
            } else {
                match := headers_re.FindAllString(line, -1)

                if len(match) > 0 {
                    header := strings.Split(match[0], ": ")

                    request.Headers[header[0]] = header[1]
                }
            }
        }
    }
}

func sendOutput(c chan *HttpRequest, quite chan int) {
    serverAddr, err := net.ResolveUDPAddr("udp4", settings.replayAddress)
    conn, err := net.DialUDP("udp", nil, serverAddr)

    defer conn.Close()

    if err != nil {
        log.Fatal("Connection error", err)
    }

    for {
        select {
        case request := <-c:
            fmt.Println("Request:", request.Url)

            msg := bytes.Buffer{}

            enc := gob.NewEncoder(&msg)
            err := enc.Encode(request)

            conn.Write(msg.Bytes())

            if err != nil {
                log.Println("encode error:", err)
            }

        case <-quite:
            conn.Close()
            return
        }
    }
}

func Run() {
    fmt.Println("Settings:", settings)

    if !strings.Contains(settings.replayAddress, ":") {
        settings.replayAddress = settings.replayAddress + ":28020"
    }

    cmd := exec.Command("tcpdump", "-vv", "-A", "-i", settings.networkInterface, "port "+strconv.Itoa(settings.port))
    //cmd := exec.Command("ls", "-al")

    stdout, _ := cmd.StdoutPipe()
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        log.Fatal(err)
    }

    c := make(chan *HttpRequest)
    err := make(chan int)

    go readOutput(stdout, c, err)
    go sendOutput(c, err)

    if err := cmd.Wait(); err != nil {
        flag.Usage()
    }
}

func init() {
    if len(os.Args) > 1 && os.Args[1] != "listen" {
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
