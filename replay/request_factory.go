package replay

import (
	"net/http"
)

// Userd for transfering Request info between Listener and Replay server
type HttpRequest struct {
	Tag     string            // Not used yet
	Method  string            // Right now only 'GET'
	Url     string            // Request URL
	Headers map[string]string // Request Headers
}

type HttpResponse struct {
	host *ForwardHost
	req  *HttpRequest
	resp *http.Response
	err  error
}

// Class for processing requests
//
// Basic workflow:
//
// 1. When request added via Add() it get pushed to `responses` chan
// 2. handleRequest() listen for `responses` chan and decide where request should be forwarded, and apply rate-limit if needed
// 3. sendRequest() forwards request and returns response info to `responses` chan
// 4. handleRequest() listen for `response` channel and updates stats
type RequestFactory struct {
	responses chan *HttpResponse
	requests  chan *HttpRequest
}

// RequestFactory contstuctor
// One created, it starts listening for incoming requests: requests channel
func NewRequestFactory() (factory *RequestFactory) {
	factory = &RequestFactory{}
	factory.responses = make(chan *HttpResponse)
	factory.requests = make(chan *HttpRequest)

	go factory.handleRequests()

	return
}

// Forward http request to given host
func (f *RequestFactory) sendRequest(host *ForwardHost, request *HttpRequest) {
	var req *http.Request

	client := &http.Client{}

	req, err := http.NewRequest("GET", host.Url+request.Url, nil)

	// Forwarded request should have same headers
	for key, value := range request.Headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)

	if err == nil {
		defer resp.Body.Close()
	}

	f.responses <- &HttpResponse{host, request, resp, err}
}

// Handle incoming requests, and they responses
func (f *RequestFactory) handleRequests() {
	hosts := Settings.ForwardedHosts()

	for {
		select {
		case req := <-f.requests:
			for _, host := range hosts {
				// Ensure that we have actual stats for given timestamp
				host.Stat.Touch()

				if host.Limit == 0 || host.Stat.Count < host.Limit {
					// Increment Stat.Count
					host.Stat.IncReq()

                    Debug("GET ",host.Url + req.Url)

					go f.sendRequest(host, req)
				}
			}
		case resp := <-f.responses:
			// Increment returned http code stats, and elapsed time
			resp.host.Stat.IncResp(resp)
		}
	}
}

// Add request to channel for further processing
func (f *RequestFactory) Add(request *HttpRequest) {
	f.requests <- request
}
