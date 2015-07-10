package main

import (
	"bytes"
	"crypto/rand"
	"io"
	"net/http"
	"sync"
	"testing"
    "strings"
    "github.com/buger/gor/proto"
    "encoding/hex"
)

// Simple service that generate token on request, and require this token for accesing to secure area
func NewFakeSecureService(wg *sync.WaitGroup) string {
	active_tokens := make([]string, 0)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
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
            } else {
                w.WriteHeader(http.StatusForbidden)
            }
		}

		wg.Done()
	})

    address := strings.Replace(listener.Addr().String(), "[::]", "127.0.0.1", -1)
	return address
}

func TestFakeSecureService(t *testing.T) {
    var resp, token []byte

	wg := new(sync.WaitGroup)

	addr := NewFakeSecureService(wg)

	wg.Add(3)

    client := NewHTTPClient("http://" + addr, &HTTPClientConfig{Debug: true})
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

func TestMiddleware(t *testing.T) {
    var resp, token []byte

    wg := new(sync.WaitGroup)

    from := NewFakeSecureService(wg)
    to := NewFakeSecureService(wg)

    quit := make(chan int)

    // Catch traffic from one service
    input := NewRAWInput(from)

    // And redirect to another
    output := NewHTTPOutput(to, &HTTPOutputConfig{})

    Plugins.Inputs = []io.Reader{input}
    Plugins.Outputs = []io.Writer{output}

    // Start Gor
    go Start(quit)

    // Should receive 2 requests from original + 2 from replayed
    wg.Add(4)

    client := NewHTTPClient("http://" + from, &HTTPClientConfig{Debug: true})

    // Sending traffic to original service
    resp, _ = client.Get("/token")
    token = proto.Body(resp)

    resp, _ = client.Get("/secure?token=" + string(token))
    if !bytes.Equal(proto.Status(resp), []byte("202")) {
        t.Error("Valid token should return 202:", proto.Status(resp))
    }

    wg.Wait()
    close(quit)
}
