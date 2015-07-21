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
)

type Middleware struct {
	command string

	data chan []byte

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
	go m.copy(m.Stdin, plugin)
}

func (m *Middleware) copy(to io.Writer, from io.Reader) {
	buf := make([]byte, 5*1024*1024)
	dst := make([]byte, len(buf)*2)

	for {
		nr, _ := from.Read(buf)
		if nr > 0 && len(buf) > nr {
			hex.Encode(dst, buf[0:nr])
			to.Write(dst[0 : nr*2])
			to.Write([]byte("\n"))
		}
	}
}

func (m *Middleware) read(from io.Reader) {
	buf := make([]byte, 5*1024*1024)

	scanner := bufio.NewScanner(from)

	for scanner.Scan() {
		bytes := scanner.Bytes()
		hex.Decode(buf, bytes)

		Debug("Received:", buf[0:len(bytes)/2])

		m.data <- buf[0 : len(bytes)/2]
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
