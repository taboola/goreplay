package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

type HTTPInput struct {
	data     chan []byte
	address  string
	listener net.Listener
}

func NewHTTPInput(address string) (i *HTTPInput) {
	i = new(HTTPInput)
	i.data = make(chan []byte)
	i.address = address

	i.listen(address)

	return
}

func (i *HTTPInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	return len(buf), nil
}

func (i *HTTPInput) handler(w http.ResponseWriter, r *http.Request) {
	buf, _ := httputil.DumpRequest(r, true)

	i.data <- buf

	http.Error(w, http.StatusText(200), 200)
}

func (i *HTTPInput) listen(address string) {
	var err error

	mux := http.NewServeMux()

	mux.HandleFunc("/", i.handler)

	i.listener, err = net.Listen("tcp", address)
	if err != nil {
		log.Fatal("HTTP input listener failure:", err)
	}

	go func() {
		err = http.Serve(i.listener, mux)
		if err != nil {
			log.Fatal("HTTP input serve failure:", err)
		}
	}()
}

func (i *HTTPInput) String() string {
	return "HTTP input: " + i.address
}
