package gor

import (
	"io"
	"net/http"
	"sync"
	"testing"
)

func TestRAWInput(t *testing.T) {
	startHTTP := func(addr string, cb func(*http.Request)) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cb(r)
		})

		go http.ListenAndServe(addr, handler)
	}

	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewRAWInput("127.0.0.1:50004")
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	startHTTP("127.0.0.1:50004", func(req *http.Request) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		http.Get("http://127.0.0.1:50004")
	}

	wg.Wait()

	close(quit)
}
