package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"github.com/buger/gor/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeServiceCb func(string, int, []byte)

// Simple service that generate token on request, and require this token for accesing to secure area
func NewFakeSecureService(wg *sync.WaitGroup, cb fakeServiceCb) string {
	active_tokens := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		Debug("Received request: " + req.URL.String())

		switch req.URL.Path {
		case "/token":
			// Generate random token
			token_length := 10
			buf := make([]byte, token_length)
			rand.Read(buf)
			token := hex.EncodeToString(buf)
			active_tokens = append(active_tokens, token)

			w.Write([]byte(token))

			cb(req.URL.Path, 200, []byte(token))
		case "/secure":
			token := req.URL.Query().Get("token")
			token_found := false

			for _, t := range active_tokens {
				if t == token {
					token_found = true
					break
				}
			}

			if token_found {
				w.WriteHeader(http.StatusAccepted)
				cb(req.URL.Path, 202, []byte(nil))
			} else {
				w.WriteHeader(http.StatusForbidden)
				cb(req.URL.Path, 403, []byte(nil))
			}
		}

		wg.Done()
	}))

	address := strings.Replace(server.Listener.Addr().String(), "[::]", "127.0.0.1", -1)
	return address
}

func TestFakeSecureService(t *testing.T) {
	var resp, token []byte

	wg := new(sync.WaitGroup)

	addr := NewFakeSecureService(wg, func(path string, status int, resp []byte) {

	})

	wg.Add(3)

	client := NewHTTPClient("http://"+addr, &HTTPClientConfig{Debug: true})
	resp, _ = client.Get("/token")
	token = proto.Body(resp)

	// Right token
	resp, _ = client.Get("/secure?token=" + string(token))
	if !bytes.Equal(proto.Status(resp), []byte("202")) {
		t.Error("Valid token should return status 202:", string(proto.Status(resp)))
	}

	// Wrong tokens forbidden
	resp, _ = client.Get("/secure?token=wrong")
	if !bytes.Equal(proto.Status(resp), []byte("403")) {
		t.Error("Wrong token should returns status 403:", string(proto.Status(resp)))
	}

	wg.Wait()
}

func TestEchoMiddleware(t *testing.T) {
	wg := new(sync.WaitGroup)

	from := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
	}))
	to := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Done()
	}))

	quit := make(chan int)

	// Catch traffic from one service
	input := NewRAWInput(from.Listener.Addr().String(), testRawExpire, true)

	// And redirect to another
	output := NewHTTPOutput(to.URL, &HTTPOutputConfig{Debug: true})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}
	Settings.middleware = "./examples/echo_modifier.sh"

	// Start Gor
	go Start(quit)

	time.Sleep(time.Millisecond)

	// Should receive 2 requests from original + 2 from replayed
	wg.Add(4)

	client := NewHTTPClient(from.URL, &HTTPClientConfig{Debug: false})

	// Request should be echoed
	client.Get("/")
	client.Get("/")

	wg.Wait()
	close(quit)
	Settings.middleware = ""

	time.Sleep(100 * time.Millisecond)
}

func TestTokenMiddleware(t *testing.T) {
	var resp, token []byte

	wg := new(sync.WaitGroup)

	from := NewFakeSecureService(wg, func(path string, status int, tok []byte) {
	})
	to := NewFakeSecureService(wg, func(path string, status int, tok []byte) {
		switch path {
		case "/token":
			if bytes.Equal(token, tok) {
				t.Error("Tokens should not match")
			}
		case "/secure":
			if status != 202 {
				// t.Error("Server should receive valid rewritten token")
			}
		}
	})

	quit := make(chan int)

	// Catch traffic from one service
	input := NewRAWInput(from, testRawExpire, true)

	// And redirect to another
	output := NewHTTPOutput(to, &HTTPOutputConfig{})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}
	Settings.middleware = "./examples/echo_modifier.sh"

	// Start Gor
	go Start(quit)

	time.Sleep(time.Millisecond)

	// Should receive 2 requests from original + 2 from replayed
	wg.Add(4)

	client := NewHTTPClient("http://"+from, &HTTPClientConfig{Debug: true})

	// Sending traffic to original service
	resp, _ = client.Get("/token")
	token = proto.Body(resp)

	resp, _ = client.Get("/secure?token=" + string(token))
	if !bytes.Equal(proto.Status(resp), []byte("202")) {
		t.Error("Valid token should return 202:", proto.Status(resp))
	}

	wg.Wait()
	close(quit)
	Settings.middleware = ""
}
