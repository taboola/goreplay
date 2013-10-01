package replay

import (
	"container/ring"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// HttpResponse contains a host, a http request,
// a http response and an error
type HttpResponse struct {
	host *ForwardHost
	req  *http.Request
	resp *http.Response
	err  error
}

// RequestFactory processes requests
//
// Basic workflow:
//
// 1. When request added via Add() it get pushed to `responses` chan
// 2. handleRequest() listen for `responses` chan and decide where request should be forwarded, and apply rate-limit if needed
// 3. sendRequest() forwards request and returns response info to `responses` chan
// 4. handleRequest() listen for `response` channel and updates stats
type RequestFactory struct {
	c_responses chan *HttpResponse
	c_requests  chan *http.Request

	reqBuf        *ring.Ring
	reqBufForSend *ring.Ring
}

// NewRequestFactory returns a RequestFactory pointer
// One created, it starts listening for incoming requests: requests channel
func NewRequestFactory() (factory *RequestFactory) {
	factory = &RequestFactory{}
	factory.c_responses = make(chan *HttpResponse)
	factory.c_requests = make(chan *http.Request)
	factory.reqBuf = ring.New(100)
	factory.reqBufForSend = factory.reqBuf

	go factory.handleRequests()
	go factory.sendRequests()

	return
}

// customCheckRedirect disables redirects https://github.com/buger/gor/pull/15
func customCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 0 {
		return errors.New("stopped after 2 redirects")
	}
	return nil
}

// sendRequest forwards http request to a given host
func (f *RequestFactory) sendRequest(host *ForwardHost, request *http.Request) {
	client := &http.Client{
		CheckRedirect: customCheckRedirect,
	}

	// Change HOST of original request
	URL := host.Url + request.URL.Path + "?" + request.URL.RawQuery

	request.RequestURI = ""
	request.URL, _ = url.ParseRequestURI(URL)

	Debug("Sending request:", host.Url, request)

	resp, err := client.Do(request)

	if err == nil {
		defer resp.Body.Close()
	} else {
		Debug("Request error:", err)
	}

	f.c_responses <- &HttpResponse{host, request, resp, err}
}

// handleRequests and their responses
func (f *RequestFactory) handleRequests() {
	hosts := Settings.ForwardedHosts()

	for {
		select {
		case req := <-f.c_requests:
			f.reqBuf.Value = req
			f.reqBuf = f.reqBuf.Next()

			for _, host := range hosts {
				if host.Limit == 0 {
					go f.sendRequest(host, req)
				}
			}
		case resp := <-f.c_responses:
			// Increment returned http code stats, and elapsed time
			resp.host.Stat.IncResp(resp)
		}
	}
}

func (f *RequestFactory) sendRequests() {
	hosts := Settings.ForwardedHosts()

	for _, host := range hosts {
		// Ensure that we have actual stats for given timestamp
		if host.Limit != 0 {
			go func() {
				for {
					host.Stat.Touch()
					// Increment Stat.Count
					host.Stat.IncReq()

					req, err := f.getReqFromBuf(0)

					if err == nil {
						go f.sendRequest(host, req)
					} else {
						fmt.Println("Request not found")
					}

					delay := float64(1) / float64(host.Limit) * float64(time.Second)

					fmt.Println(int(time.Duration(delay) / time.Millisecond))
					time.Sleep(time.Duration(delay))
				}
			}()
		}
	}
}

func (f *RequestFactory) getReqFromBuf(retries int) (*http.Request, error) {
	f.reqBufForSend = f.reqBufForSend.Prev()

	if f.reqBufForSend.Value != nil {
		return f.reqBufForSend.Value.(*http.Request), nil
	} else {
		if retries <= f.reqBufForSend.Len() {
			return f.getReqFromBuf(retries + 1)
		} else {
			return nil, errors.New("Requests not found")
		}
	}
}

// Add request to channel for further processing
func (f *RequestFactory) Add(request *http.Request) {
	f.c_requests <- request
}
