package main

import (
	_ "bufio"
	"bytes"
	"crypto/rand"
	_ "io"
	"io/ioutil"
	_ "log"
	_ "net"
	"net/http"
	"sync"
	"testing"
)

// Simple service that generate token on request, and require this token for accesing to secure area
func NewFakeSecureService(wg *sync.WaitGroup) string {
	active_tokens := make([][]byte, 0)

	listener := startHTTP(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/token":
			// Generate random token
			token_length := 10
			token := make([]byte, token_length)
			rand.Read(token)

			w.Write(token)

			active_tokens = append(active_tokens, token)
		case "/secure":
			token := []byte(req.URL.Query().Get("token"))

			for _, t := range active_tokens {
				if bytes.Equal(t, token) {
					w.WriteHeader(http.StatusAccepted)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
			}
		}

		wg.Done()
	})

	return "http://" + listener.Addr().String()
}

func TestFakeSecureService(t *testing.T) {
	var resp *http.Response

	wg := new(sync.WaitGroup)

	addr := NewFakeSecureService(wg)

	wg.Add(3)

	resp, _ = http.Get(addr + "/token")
	token, _ := ioutil.ReadAll(resp.Body)

	// Right token
	resp, _ = http.Get(addr + "/secure?token=" + string(token))
	if resp.StatusCode != http.StatusAccepted {
		t.Error("Valid token should returns wrong status:", resp.StatusCode)
	}

	// Wrong tokens forbidden
	resp, _ = http.Get(addr + "/secure?token=wrong")
	if resp.StatusCode != http.StatusForbidden {
		t.Error("Wrong tokens should be forbidden, instead:", resp.StatusCode)
	}

	wg.Wait()
}
