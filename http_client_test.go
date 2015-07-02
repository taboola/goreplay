package main

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"sync"
	"testing"
	_ "time"
)

func TestHTTPClientURLPort(t *testing.T) {
	c1 := NewHTTPClient("http://example.com", &HTTPClientConfig{})
	if c1.baseURL.String() != "http://example.com:80" {
		t.Error("Sould add 80 port for http:", c1.baseURL.String())
	}

	c2 := NewHTTPClient("https://example.com", &HTTPClientConfig{})
	if c2.baseURL.String() != "https://example.com:443" {
		t.Error("Sould add 443 port for https:", c2.baseURL.String())
	}

	c3 := NewHTTPClient("https://example.com:1", &HTTPClientConfig{})
	if c3.baseURL.String() != "https://example.com:1" {
		t.Error("Sould use specified port:", c3.baseURL.String())
	}

	c4 := NewHTTPClient("example.com", &HTTPClientConfig{})
	if c4.baseURL.String() != "http://example.com:80" {
		t.Error("Sould add default protocol:", c4.baseURL.String())
	}
}

func TestHTTPClientSend(t *testing.T) {
	wg := new(sync.WaitGroup)

	GET_payload := []byte("GET / HTTP/1.1\r\n\r\n")

	// Post request terminates by reading Content-Length without double CRLF
	POST_payload := []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

	// Chunked requests terminated with double CRLF
	POST_CHUNKED_payload := []byte("POST / HTTP/1.1\r\nHost: www.w3.org\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nWiki\r\n5\r\npedia\r\ne\r\n in\r\n\r\nchunks.\r\n0\r\n\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" {
			defer r.Body.Close()
			body, _ := ioutil.ReadAll(r.Body)

			if len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked" {
				if string(body) != "Wikipedia in\r\n\r\nchunks." {
					t.Error("Wrong POST body:", body, string(body))
				}
			} else {
				if string(body) != "a=1&b=2" {
					buf, _ := httputil.DumpRequest(r, true)
					t.Error("Wrong POST body:", string(body), string(buf))
				}
			}
		}

		wg.Done()
	}))

	client := NewHTTPClient(server.URL, &HTTPClientConfig{})

	wg.Add(4)
	client.Send(POST_payload)
	client.Send(GET_payload)
	client.Send(POST_CHUNKED_payload)
	client.Send(POST_payload)

	wg.Wait()
}

func TestHTTPClientHTTPSSend(t *testing.T) {
	wg := new(sync.WaitGroup)

	GET_payload := []byte("GET / HTTP/1.1\r\n\r\n")

	// Post request terminates by reading Content-Length without double CRLF
	POST_payload := []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

	// Chunked requests terminated with double CRLF
	POST_CHUNKED_payload := []byte("POST / HTTP/1.1\r\nHost: www.w3.org\r\nTransfer-Encoding: chunked\r\n\r\n4\r\nWiki\r\n5\r\npedia\r\ne\r\n in\r\n\r\nchunks.\r\n0\r\n\r\n")

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" {
			defer r.Body.Close()
			body, _ := ioutil.ReadAll(r.Body)

			if len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked" {
				if string(body) != "Wikipedia in\r\n\r\nchunks." {
					t.Error("Wrong POST body:", body, string(body))
				}
			} else {
				if string(body) != "a=1&b=2" {
					buf, _ := httputil.DumpRequest(r, true)
					t.Error("Wrong POST body:", string(body), string(buf))
				}
			}
		}

		wg.Done()
	}))

	client := NewHTTPClient(server.URL, &HTTPClientConfig{})

	wg.Add(4)
	client.Send(GET_payload)
	client.Send(POST_payload)
	client.Send(POST_CHUNKED_payload)
	client.Send(POST_payload)

	wg.Wait()
}

func TestHTTPClientServerInstantDisconnect(t *testing.T) {
	wg := new(sync.WaitGroup)

	GET_payload := []byte("GET / HTTP/1.1\r\n\r\n")

	ln, _ := net.Listen("tcp", ":0")

	go func() {
		for {
			conn, _ := ln.Accept()
			conn.Close()

			wg.Done()
		}
	}()

	client := NewHTTPClient(ln.Addr().String(), &HTTPClientConfig{})

	wg.Add(2)
	client.Send(GET_payload)
	client.Send(GET_payload)

	wg.Wait()
}

func TestHTTPClientServerNoKeepAlive(t *testing.T) {
	wg := new(sync.WaitGroup)

	GET_payload := []byte("GET / HTTP/1.1\r\n\r\n")

	ln, _ := net.Listen("tcp", ":0")

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				// handle error
			}

			buf := make([]byte, 4096)
			reqLen, err := conn.Read(buf)
			if err != nil {
				t.Error("Error reading:", err.Error())
			}
			Debug("Received: ", string(buf[0:reqLen]))
			conn.Write([]byte("OK"))

			// No keep-alive connections
			conn.Close()

			wg.Done()
		}
	}()

	client := NewHTTPClient(ln.Addr().String(), &HTTPClientConfig{})

	wg.Add(2)
	client.Send(GET_payload)
	client.Send(GET_payload)

	wg.Wait()
}

func TestHTTPClientRedirect(t *testing.T) {
	wg := new(sync.WaitGroup)

	GET_payload := []byte("GET / HTTP/1.1\r\n\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/" {
			http.Redirect(w, r, "/new", 301)
		}

		wg.Done()
	}))

	client := NewHTTPClient(server.URL, &HTTPClientConfig{FollowRedirects: 1, Debug: false})

	// Should do 2 queries
	wg.Add(2)
	client.Send(GET_payload)

	wg.Wait()
}

func TestHTTPClientRedirectLimit(t *testing.T) {
	wg := new(sync.WaitGroup)

	GET_payload := []byte("GET / HTTP/1.1\r\n\r\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/" {
			http.Redirect(w, r, "/r1", 301)
		}

		if r.URL.Path == "/r1" {
			http.Redirect(w, r, "/r2", 301)
		}

		if r.URL.Path == "/r2" {
			http.Redirect(w, r, "/new", 301)
		}

		wg.Done()
	}))

	client := NewHTTPClient(server.URL, &HTTPClientConfig{FollowRedirects: 2, Debug: false})

	// Have 3 redirects + 1 GET, but should do only 2 redirects + GET
	wg.Add(3)
	client.Send(GET_payload)

	wg.Wait()
}