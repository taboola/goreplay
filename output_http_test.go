package main

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	_ "net/http/httputil"
	"sync"
	"testing"
	"time"
)

func startHTTP(cb func(http.ResponseWriter, *http.Request)) net.Listener {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cb(w, r)
	})

	listener, _ := net.Listen("tcp", ":0")

	go http.Serve(listener, handler)

	return listener
}

func TestHTTPOutput(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewTestInput()

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("User-Agent") != "Gor" {
			t.Error("Wrong header")
		}

		if req.Method == "OPTIONS" {
			t.Error("Wrong method")
		}

		if req.Method == "POST" {
			defer req.Body.Close()
			body, _ := ioutil.ReadAll(req.Body)

			if string(body) != "a=1&b=2" {
				t.Error("Wrong POST body:", string(body))
			}
		}

		wg.Done()
	})

	headers := HTTPHeaders{HTTPHeader{"User-Agent", "Gor"}}
	methods := HTTPMethods{[]byte("GET"), []byte("PUT"), []byte("POST")}
	Settings.modifierConfig = HTTPModifierConfig{headers: headers, methods: methods}

	output := NewHTTPOutput(listener.Addr().String(), &HTTPOutputConfig{})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(2) // OPTIONS should be ignored
		input.EmitPOST()
		input.EmitOPTIONS()
		input.EmitGET()
	}

	wg.Wait()

	close(quit)

	Settings.modifierConfig = HTTPModifierConfig{}
}

func TestHTTPOutputKeepOriginalHost(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewTestInput()

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		if req.Host != "custom-host.com" {
			t.Error("Wrong header", req.Host)
		}

		wg.Done()
	})

	headers := HTTPHeaders{HTTPHeader{"Host", "custom-host.com"}}
	Settings.modifierConfig = HTTPModifierConfig{headers: headers}

	output := NewHTTPOutput(listener.Addr().String(), &HTTPOutputConfig{Debug: false, OriginalHost: true})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	wg.Add(1)
	input.EmitGET()

	wg.Wait()

	close(quit)

	Settings.modifierConfig = HTTPModifierConfig{}
}

func TestOutputHTTPSSL(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	// Origing and Replay server initialization
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
	}))

	input := NewTestInput()
	output := NewHTTPOutput(server.URL, &HTTPOutputConfig{})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	wg.Add(2)

	input.EmitPOST()
	input.EmitGET()

	wg.Wait()
	close(quit)
}

func BenchmarkHTTPOutput(b *testing.B) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(50 * time.Millisecond)
		wg.Done()
	})

	input := NewTestInput()
	output := NewHTTPOutput(listener.Addr().String(), &HTTPOutputConfig{})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		input.EmitPOST()
	}

	wg.Wait()

	close(quit)
}
