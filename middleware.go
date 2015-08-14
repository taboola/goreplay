package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Middleware struct {
	command string

	data chan []byte

	mu sync.Mutex

	Stdin  io.Writer
	Stdout io.Reader
}

func NewMiddleware(command string) *Middleware {
	m := new(Middleware)
	m.command = command
	m.data = make(chan []byte, 1000)

	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...)

	m.Stdout, _ = cmd.StdoutPipe()
	m.Stdin, _ = cmd.StdinPipe()
	cmd.Stderr = os.Stderr

	go m.read(m.Stdout)

	go func() {
		err := cmd.Start()

		if err != nil {
			log.Fatal(err)
		}

		cmd.Wait()
	}()

	return m
}

func (m *Middleware) ReadFrom(plugin io.Reader) {
	Debug("[MIDDLEWARE-MASTER] Starting reading from", plugin)
	go m.copy(m.Stdin, plugin)
}

func (m *Middleware) copy(to io.Writer, from io.Reader) {
	buf := make([]byte, 5*1024*1024)
	dst := make([]byte, len(buf)*2)

	for {
		nr, _ := from.Read(buf)
		if nr > 0 && len(buf) > nr {

			hex.Encode(dst, buf[0:nr])
			dst[nr*2] = '\n'

			m.mu.Lock()
			to.Write(dst[0 : nr*2+1])
			m.mu.Unlock()

			if Settings.debug {
				Debug("[MIDDLEWARE-MASTER] Sending:", string(buf[0:nr]), "From:", from)
			}
		}
	}
}

func (m *Middleware) read(from io.Reader) {
	scanner := bufio.NewScanner(from)

	for scanner.Scan() {
		bytes := scanner.Bytes()
		buf := make([]byte, len(bytes)/2)
		if _, err := hex.Decode(buf, bytes); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to decode input payload", err, len(bytes))
		}

		if Settings.debug {
			Debug("[MIDDLEWARE-MASTER] Received:", string(buf))
		}

		// We should accept only request payloads
		if buf[0] == '1' {
			m.data <- buf
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Traffic modifier command failed:", err)
	}

	return
}

func (m *Middleware) Read(data []byte) (int, error) {
	Debug("Trying to read channel!")
	buf := <-m.data
	copy(data, buf)

	return len(buf), nil
}

func (m *Middleware) String() string {
	return fmt.Sprintf("Modifying traffic using '%s' command", m.command)
}
