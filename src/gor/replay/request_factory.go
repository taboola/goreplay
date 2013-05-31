package replay

import (
	"net/http"
	"fmt"
)

type HttpRequest struct {
	Tag     string
	Method  string
	Url     string
	Headers map[string]string
}

type HttpResponse struct {
	host *ForwardHost
	req  *HttpRequest
	resp *http.Response
	err  error
}

type RequestFactory struct {
	responses chan *HttpResponse
	requests  chan *HttpRequest
}

func NewRequestFactory() (factory *RequestFactory) {
	factory = &RequestFactory{}
	factory.responses = make(chan *HttpResponse)
	factory.requests = make(chan *HttpRequest)

	go factory.handleRequests()

	return
}

func (f *RequestFactory) sendRequest(host *ForwardHost, request *HttpRequest) {
	var req *http.Request

	client := &http.Client{}

	req, err := http.NewRequest("GET", host.Url+request.Url, nil)

	for key, value := range request.Headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)

	defer resp.Body.Close()

	f.responses <- &HttpResponse{host, request, resp, err}
}

func (f *RequestFactory) handleRequests() {
	hosts := settings.ForwardedHosts()

	for {
		select {
		case req := <- f.requests:
			for _, host := range hosts {
				host.Stat.Touch()

				if host.Limit == 0 || host.Stat.Count < host.Limit {
					host.Stat.IncReq()

					fmt.Println("Sending request")
					go f.sendRequest(host, req)
				} else {
					fmt.Println("Throttling for host:", host.Url, host.Stat.Count, host.Limit)
				}
			}
		case resp := <- f.responses:
			resp.host.Stat.IncResp(resp)
		}
	}
}

func (f *RequestFactory) Add(request *HttpRequest) {
	f.requests <- request
}
