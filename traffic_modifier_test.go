package main

import (
	_ "bufio"
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	_ "log"
	_ "net"
	"net/http"
	"sync"
	"testing"
    "strings"
)

// Simple service that generate token on request, and require this token for accesing to secure area
func NewFakeSecureService(wg *sync.WaitGroup) string {
	active_tokens := make([][]byte, 0)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
        Debug("Received request: " + req.URL.String())

		switch req.URL.Path {
		case "/token":
			// Generate random token
			token_length := 10
			token := make([]byte, token_length)
			rand.Read(token)
			active_tokens = append(active_tokens, token)

            w.Write(token)
		case "/secure":
			token := []byte(req.URL.Query().Get("token"))
            token_found := false

			for _, t := range active_tokens {
				if bytes.Equal(t, token) {
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
	var resp *http.Response

	wg := new(sync.WaitGroup)

	addr := NewFakeSecureService(wg)

	wg.Add(3)

	resp, _ = http.Get("http://" + addr + "/token")
	token, _ := ioutil.ReadAll(resp.Body)

	// Right token
	resp, _ = http.Get("http://" + addr + "/secure?token=" + string(token))
	if resp.StatusCode != http.StatusAccepted {
		t.Error("Valid token should returns wrong status:", resp.StatusCode)
	}

	// Wrong tokens forbidden
	resp, _ = http.Get("http://" + addr + "/secure?token=wrong")
	if resp.StatusCode != http.StatusForbidden {
		t.Error("Wrong tokens should be forbidden, instead:", resp.StatusCode)
	}

	wg.Wait()
}

func TestTrafficModifier(t *testing.T) {
    var resp *http.Response

    wg := new(sync.WaitGroup)

    from := NewFakeSecureService(wg)
    to := NewFakeSecureService(wg)

    quit := make(chan int)

    // Catch traffic from one service
    input := NewRAWInput(from)

    // And redirect to another
    headers := HTTPHeaders{HTTPHeader{"User-Agent", "Gor"}}
    methods := HTTPMethods{"GET", "PUT", "POST"}
    output := NewHTTPOutput(to, headers, methods, HTTPUrlRegexp{}, HTTPHeaderFilters{}, HTTPHeaderHashFilters{}, "", UrlRewriteMap{}, 0)

    Plugins.Inputs = []io.Reader{input}
    Plugins.Outputs = []io.Writer{output}

    // Start Gor
    go Start(quit)

    // Should receive 2 requests from original + 2 from replayed
    wg.Add(4)

    // Sending traffic to original service
    resp, _ = http.Get("http://" + from + "/token")
    token, _ := ioutil.ReadAll(resp.Body)

    resp, _ = http.Get("http://" + from + "/secure?token=" + string(token))
    if resp.StatusCode != http.StatusAccepted {
        t.Error("Valid token should returns wrong status:", resp.StatusCode)
    }

    wg.Wait()
    close(quit)
}
