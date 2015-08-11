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
	"time"
)

const testRawExpire = time.Millisecond * 100

func TestRAWInput(t *testing.T) {

	wg := new(sync.WaitGroup)
	quit := make(chan int)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {})

	input := NewRAWInput(listener.Addr().String(), testRawExpire, false)
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	address := strings.Replace(listener.Addr().String(), "[::]", "127.0.0.1", -1)

	client := NewHTTPClient(address, &HTTPClientConfig{})

	time.Sleep(time.Millisecond)

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

	fileContent, _ := ioutil.ReadFile("README.md")

	// Origing and Replay server initialization
	origin := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		ioutil.ReadAll(req.Body)

		wg.Done()
	})

	originAddr := strings.Replace(origin.Addr().String(), "[::]", "127.0.0.1", -1)

	input := NewRAWInput(originAddr, testRawExpire, false)

	// We will use it to get content of raw HTTP request
	testOutput := NewTestOutput(func(data []byte) {
		if strings.Contains(string(data), "Expect: 100-continue") {
			t.Error("Should not contain 100-continue header")
		}
		wg.Done()
	})

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		body, _ := ioutil.ReadAll(req.Body)

		if !bytes.Equal(body, fileContent) {
			buf, _ := httputil.DumpRequest(req, true)
			t.Error("Wrong POST body:", string(buf))
		}

		wg.Done()
	})
	replayAddr := listener.Addr().String()

	httpOutput := NewHTTPOutput(replayAddr, &HTTPOutputConfig{})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{testOutput, httpOutput}

	go Start(quit)

	wg.Add(3)
	curl := exec.Command("curl", "http://"+originAddr, "--data-binary", "@README.md")
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

	fileContent, _ := ioutil.ReadFile("README.md")

	// Origing and Replay server initialization
	origin := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		ioutil.ReadAll(req.Body)

		wg.Done()
	})

	originAddr := strings.Replace(origin.Addr().String(), "[::]", "127.0.0.1", -1)

	input := NewRAWInput(originAddr, testRawExpire, false)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		body, _ := ioutil.ReadAll(req.Body)

		if !bytes.Equal(body, fileContent) {
			buf, _ := httputil.DumpRequest(req, true)
			t.Error("Wrong POST body:", string(buf))
		}

		wg.Done()
	})
	replayAddr := listener.Addr().String()

	httpOutput := NewHTTPOutput(replayAddr, &HTTPOutputConfig{Debug: true})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{httpOutput}

	go Start(quit)

	wg.Add(2)

	curl := exec.Command("curl", "http://"+originAddr, "--header", "Transfer-Encoding: chunked", "--data-binary", "@README.md")
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
	dd := exec.Command("dd", "if=/dev/urandom", "of=/tmp/large", "bs=1KB", "count=100")
	err := dd.Run()
	if err != nil {
		log.Fatal("dd error:", err)
	}

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		body, _ := ioutil.ReadAll(req.Body)

		if len(body) != 100*1000 {
			t.Error("File size should be 1mb:", len(body))
		}

		wg.Done()
	}))
	originAddr := strings.Replace(origin.Listener.Addr().String(), "[::]", "127.0.0.1", -1)

	input := NewRAWInput(originAddr, testRawExpire, false)

	replay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(w, req.Body, 1*1024*1024)
		buf := make([]byte, 1*1024*1024)
		n, _ := req.Body.Read(buf)
		body := buf[0:n]

		if len(body) != 100*1000 {
			t.Error("File size should be 100000 bytes:", len(body))
		}

		wg.Done()
	}))
	defer replay.Close()

	httpOutput := NewHTTPOutput(replay.URL, &HTTPOutputConfig{Debug: false})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{httpOutput}

	go Start(quit)

	wg.Add(2)
	curl := exec.Command("curl", "http://"+originAddr, "--header", "Transfer-Encoding: chunked", "--data-binary", "@/tmp/large")
	err = curl.Run()
	if err != nil {
		log.Fatal("curl error:", err)
	}

	wg.Wait()
	close(quit)
}

func TestInputRAWResponse(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {})

	input := NewRAWInput(listener.Addr().String(), testRawExpire, true)
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	address := strings.Replace(listener.Addr().String(), "[::]", "127.0.0.1", -1)

	client := NewHTTPClient(address, &HTTPClientConfig{})

	time.Sleep(time.Millisecond)
	go Start(quit)

	for i := 0; i < 100; i++ {
		// 2 because we track both request and response
		wg.Add(2)
		client.Get("/")
	}

	wg.Wait()
	close(quit)

	time.Sleep(100*time.Millisecond)
}