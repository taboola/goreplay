package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func TestRAWInput(t *testing.T) {

	wg := new(sync.WaitGroup)
	quit := make(chan int)

	listener := startHTTP(func(req *http.Request) {})

	input := NewRAWInput(listener.Addr().String())
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	address := strings.Replace(listener.Addr().String(), "[::]", "127.0.0.1", -1)

	client := NewHTTPClient(address,  &HTTPClientConfig{})

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		client.Get("/")
	}

	wg.Wait()

	close(quit)
}

func TestInputRAW100Expect(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	file_content, _ := ioutil.ReadFile("README.md")

	// Origing and Replay server initialization
	origin := startHTTP(func(req *http.Request) {
		defer req.Body.Close()
		ioutil.ReadAll(req.Body)

		wg.Done()
	})

	origin_address := strings.Replace(origin.Addr().String(), "[::]", "127.0.0.1", -1)

	input := NewRAWInput(origin_address)

	// We will use it to get content of raw HTTP request
	test_output := NewTestOutput(func(data []byte) {
		if strings.Contains(string(data), "Expect: 100-continue") {
			t.Error("Should not contain 100-continue header")
		}
		wg.Done()
	})

	listener := startHTTP(func(req *http.Request) {
		defer req.Body.Close()
		body, _ := ioutil.ReadAll(req.Body)

		if !bytes.Equal(body, file_content) {
			buf, _ := httputil.DumpRequest(req, true)
			t.Error("Wrong POST body:", string(buf))
		}

		wg.Done()
	})
	replay_address := listener.Addr().String()

	http_output := NewHTTPOutput(replay_address, &HTTPOutputConfig{})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{test_output, http_output}

	go Start(quit)

	wg.Add(3)
	curl := exec.Command("curl", "http://"+origin_address, "--data-binary", "@README.md")
	err := curl.Run()
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	close(quit)
}

func TestInputRAWChunkedEncoding(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	file_content, _ := ioutil.ReadFile("README.md")

	// Origing and Replay server initialization
	origin := startHTTP(func(req *http.Request) {
		defer req.Body.Close()
		ioutil.ReadAll(req.Body)

		wg.Done()
	})

	origin_address := strings.Replace(origin.Addr().String(), "[::]", "127.0.0.1", -1)

	input := NewRAWInput(origin_address)

	listener := startHTTP(func(req *http.Request) {
		defer req.Body.Close()
		body, _ := ioutil.ReadAll(req.Body)

		if !bytes.Equal(body, file_content) {
			buf, _ := httputil.DumpRequest(req, true)
			t.Error("Wrong POST body:", string(buf))
		}

		wg.Done()
	})
	replay_address := listener.Addr().String()

	http_output := NewHTTPOutput(replay_address, &HTTPOutputConfig{Debug: true})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{http_output}

	go Start(quit)

	wg.Add(2)

	curl := exec.Command("curl", "http://"+origin_address, "--header", "Transfer-Encoding: chunked", "--data-binary", "@README.md")
	err := curl.Run()
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()

	close(quit)
}

func TestInputRAWLargePayload(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	// Generate 200kb file
	dd := exec.Command("dd", "if=/dev/urandom", "of=/tmp/large", "bs=1KB", "count=200")
	err := dd.Run()
	if err != nil {
		log.Fatal("dd error:", err)
	}

	// Origing and Replay server initialization
	origin := startHTTP(func(req *http.Request) {
		defer req.Body.Close()
		body, _ := ioutil.ReadAll(req.Body)

		if len(body) != 200*1000 {
			t.Error("File size should be 1mb:", len(body))
		}

		wg.Done()
	})
	origin_address := strings.Replace(origin.Addr().String(), "[::]", "127.0.0.1", -1)

	input := NewRAWInput(origin_address)

	replay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(w, req.Body, 1*1024*1024)
		buf := make([]byte, 1*1024*1024)
		n, _ := req.Body.Read(buf)
		body := buf[0:n]

		if len(body) != 200*1000 {
			t.Error("File size should be 200000 bytes:", len(body))
		}

		wg.Done()
	}))
	defer replay.Close()

	http_output := NewHTTPOutput(replay.URL, &HTTPOutputConfig{Debug: false})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{http_output}

	go Start(quit)

	wg.Add(2)
	curl := exec.Command("curl", "http://"+origin_address, "--header", "Transfer-Encoding: chunked", "--data-binary", "@/tmp/large")
	err = curl.Run()
	if err != nil {
		log.Fatal("curl error:", err)
	}

	wg.Wait()
	close(quit)
}
