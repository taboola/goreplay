package gor

import (
	"bufio"
	"bytes"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type RedirectNotAllowed struct{}

func (e *RedirectNotAllowed) Error() string {
	return "Redirects not allowed"
}

// customCheckRedirect disables redirects https://github.com/buger/gor/pull/15
func customCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 0 {
		return new(RedirectNotAllowed)
	}
	return nil
}

// ParseRequest in []byte returns a http request or an error
func ParseRequest(data []byte) (request *http.Request, err error) {
	buf := bytes.NewBuffer(data)
	reader := bufio.NewReader(buf)

	request, err = http.ReadRequest(reader)

	return
}

type HTTPOutput struct {
	address string
}

func NewHTTPOutput(address string) (o *HTTPOutput) {
	o = new(HTTPOutput)

	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}

	o.address = address

	return
}

func (o *HTTPOutput) Write(data []byte) (n int, err error) {
	go o.sendRequest(data)

	return len(data), nil
}

func (o *HTTPOutput) sendRequest(data []byte) {
	request, err := ParseRequest(data)

	if err != nil {
		log.Println("Can not parse request", string(data), err)
		return
	}

	client := &http.Client{
		CheckRedirect: customCheckRedirect,
	}

	// Change HOST of original request
	URL := o.address + request.URL.Path + "?" + request.URL.RawQuery

	request.RequestURI = ""
	request.URL, _ = url.ParseRequestURI(URL)

	resp, err := client.Do(request)

	// We should not count Redirect as errors
	if _, ok := err.(*RedirectNotAllowed); ok {
		err = nil
	}

	if err == nil {
		defer resp.Body.Close()
	} else {
		log.Println("Request error:", err)
	}
}

func (o *HTTPOutput) String() string {
	return "HTTP output: " + o.address
}
