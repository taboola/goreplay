package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		res, _ := http.Get("http://" + address)
		res.Body.Close()
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

	headers := HTTPHeaders{HTTPHeader{"User-Agent", "Gor"}}
	methods := HTTPMethods{"GET", "PUT", "POST"}
	http_output := NewHTTPOutput(replay_address, headers, methods, HTTPUrlRegexp{}, HTTPHeaderFilters{}, HTTPHeaderHashFilters{}, "", UrlRewriteMap{}, 0)

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

	// We will use it to get content of raw HTTP request
	test_output := NewTestOutput(func(data []byte) {
		if strings.Contains(string(data), "Transfer-Encoding: chunked") {
			t.Error("Should not contain chunked header")
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

	headers := HTTPHeaders{HTTPHeader{"User-Agent", "Gor"}}
	methods := HTTPMethods{"GET", "PUT", "POST"}
	http_output := NewHTTPOutput(replay_address, headers, methods, HTTPUrlRegexp{}, HTTPHeaderFilters{}, HTTPHeaderHashFilters{}, "", UrlRewriteMap{}, 0)

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{test_output, http_output}

	go Start(quit)

	wg.Add(3)

	curl := exec.Command("curl", "http://"+origin_address, "--header", "Transfer-Encoding: chunked", "--data-binary", "@README.md")
	err := curl.Run()
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()

	close(quit)
}
