package main

import (
    "fmt"
    "log"
    "io"
    "os/exec"
    "os"
    "bufio"
    "encoding/hex"
)

type TrafficModifier struct {
    plugin interface{}
    command string

    data chan []byte

    Stdin io.Writer
    Stdout io.Reader
}

func NewTrafficModifier(plugin interface{}, command string) io.Reader {
    m := new(TrafficModifier)
    m.plugin = plugin
    m.command = command
    m.data = make(chan []byte)

    cmd := exec.Command(command)

    m.Stdout, _ = cmd.StdoutPipe()
    m.Stdin, _ = cmd.StdinPipe()
    cmd.Stderr = os.Stderr

    go m.copy(m.Stdin, m.plugin.(io.Reader))
    go m.read(m.Stdout)

    go func(){
        err := cmd.Start()

        if (err != nil) {
            log.Fatal(err)
        }
    }()

    defer cmd.Wait()

    return m
}

func (m *TrafficModifier) copy(to io.Writer, from io.Reader) {
    buf := make([]byte, 5*1024*1024)
    dst := make([]byte, len(buf)*2)

    for {
        nr, _ := from.Read(buf)
        if nr > 0 && len(buf) > nr {
            hex.Encode(dst, buf[0:nr])
            to.Write(dst[0:nr*2])
            to.Write([]byte("\r\n"))
        }
    }
}

func (m *TrafficModifier) read(from io.Reader) {
    buf := make([]byte, 5*1024*1024)

    scanner := bufio.NewScanner(from)

    for scanner.Scan() {
        bytes := scanner.Bytes()
        hex.Decode(buf, bytes)

        Debug("Received:", buf[0:len(bytes)/2])

        m.data <- buf[0:len(bytes)/2]
    }

    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, "Traffic modifier command failed:", err)
    }

    return
}

func (m *TrafficModifier) Read(data []byte) (int, error) {
    Debug("Trying to read channel!")
    buf := <- m.data
    copy(data, buf)

    return len(buf), nil
}


func (m *TrafficModifier) String() string {
    return fmt.Sprintf("Modifying traffic for %s using '%s' command", m.plugin, m.command)
}
