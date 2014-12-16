package main

import (
    "fmt"
    "log"
    "io"
    "os/exec"
    "os"
    "encoding/base64"
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

    cmd := exec.Command("bash", "-c", command)
    cmd.Stderr = os.Stderr

    m.Stdout, _ = cmd.StdoutPipe()
    m.Stdin, _ = cmd.StdinPipe()

    m.Stdout = base64.NewDecoder(base64.StdEncoding, m.Stdout)

    go m.copy(m.Stdin, m.plugin.(io.Reader))

    err := cmd.Run()

    if (err != nil) {
        log.Fatal(err)
    }

    return m
}

func (m *TrafficModifier) copy(to io.Writer, from io.Reader) {
    buf := make([]byte, 5*1024*1024)

    for {
        nr, er := from.Read(buf)
        if nr > 0 && len(buf) > nr {
            to.Write(base64.StdEncoding.Encode(buf))
        }
    }
}

func (m *TrafficModifier) Read(data []byte) (n int, err error) {
    n, err = m.Stdout.Read(data)

    return
}

func (m *TrafficModifier) String() string {
    return fmt.Sprintf("Modifying traffic for %s using '%s' command", m.plugin, m.command)
}
