package main

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestHTTPInput(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewHTTPInput(":0")
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	address := strings.Replace(input.listener.Addr().String(), "[::]", "127.0.0.1", -1)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		http.Get("http://" + address + "/")
	}

	wg.Wait()

	close(quit)
}
