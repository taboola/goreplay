package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type RedirectNotAllowed struct{}

func (e *RedirectNotAllowed) Error() string {
	return "Redirects not allowed"
}

// customCheckRedirect disables redirects https://github.com/buger/gor/pull/15
func (o *HTTPOutput) customCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= o.config.redirectLimit {
		return new(RedirectNotAllowed)
	}
	return nil
}

// ParseRequest in []byte returns a http request or an error
func ParseRequest(data []byte) (request *http.Request, err error) {
	var body []byte

	buf := bytes.NewBuffer(data)
	reader := bufio.NewReader(buf)

	// ReadRequest does not read POST bodies, we have to do it by ourseves
	request, err = http.ReadRequest(reader)

	if err != nil {
		return
	}

	if request.Method == "POST" {
		body, _ = ioutil.ReadAll(reader)

		bodyBuf := bytes.NewBuffer(body)

		request.Body = ioutil.NopCloser(bodyBuf)
		request.ContentLength = int64(bodyBuf.Len())
	}

	return
}

const InitialDynamicWorkers = 10

type HTTPOutputConfig struct {
	redirectLimit int

	stats bool
	workers int

	modifier HTTPModifierConfig

	elasticSearch string
}

type HTTPOutput struct {
	// Keep this as first element of struct because it guarantees 64bit
	// alignment. atomic.* functions crash on 32bit machines if operand is not
	// aligned at 64bit. See https://github.com/golang/go/issues/599
	activeWorkers int64

	address string
	limit   int
	queue   chan []byte

	needWorker chan int

	config *HTTPOutputConfig

	queueStats *GorStat

	elasticSearch *ESPlugin
}

func NewHTTPOutput(address string, config *HTTPOutputConfig) io.Writer {

	o := new(HTTPOutput)

	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}

	o.address = address
	o.config = config

	if o.config.stats {
		o.queueStats = NewGorStat("output_http")
	}

	o.queue = make(chan []byte, 100)
	o.needWorker = make(chan int, 1)

	// Initial workers count
	if o.config.workers == 0 {
		o.needWorker <- InitialDynamicWorkers
	} else {
		o.needWorker <- o.config.workers
	}

	if o.config.elasticSearch != "" {
		o.elasticSearch = new(ESPlugin)
		o.elasticSearch.Init(o.config.elasticSearch)
	}

	go o.WorkerMaster()

	return o
}

func (o *HTTPOutput) WorkerMaster() {
	for {
		new_workers := <-o.needWorker
		for i := 0; i < new_workers; i++ {
			go o.Worker()
		}

		// Disable dynamic scaling if workers poll fixed size
		if o.config.workers != 0 {
			return
		}
	}
}

func (o *HTTPOutput) Worker() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport:     tr,
		CheckRedirect: o.customCheckRedirect,
	}

	death_count := 0

	atomic.AddInt64(&o.activeWorkers, 1)

	for {
		select {
		case data := <-o.queue:
			o.sendRequest(client, data)
			death_count = 0
		case <-time.After(time.Millisecond * 100):
			// When dynamic scaling enabled workers die after 2s of inactivity
			if o.config.workers == 0 {
				death_count += 1
			} else {
				continue
			}

			if death_count > 20 {
				workersCount := atomic.LoadInt64(&o.activeWorkers)

				// At least 1 worker should be alive
				if workersCount != 1 {
					atomic.AddInt64(&o.activeWorkers, -1)
					return
				}
			}
		}
	}
}

func (o *HTTPOutput) Write(data []byte) (n int, err error) {
	buf := make([]byte, len(data))
	copy(buf, data)

	o.queue <- buf

	if o.config.stats {
		o.queueStats.Write(len(o.queue))
	}

	if o.config.workers == 0 {
		workersCount := atomic.LoadInt64(&o.activeWorkers)

		if len(o.queue) > int(workersCount) {
			o.needWorker <- len(o.queue)
		}
	}

	return len(data), nil
}

func (o *HTTPOutput) sendRequest(client *http.Client, data []byte) {
	request, err := ParseRequest(data)

	if err != nil {
		log.Println("Cannot parse request", string(data), err)
		return
	}

	if len(o.config.modifier.methods) > 0 && !o.config.modifier.methods.Contains(request.Method) {
		return
	}

	if !(o.config.modifier.urlRegexp.Good(request) && o.config.modifier.headerFilters.Good(request) && o.config.modifier.headerHashFilters.Good(request)) {
		return
	}

	// Rewrite the path as necessary
	request.URL.Path = o.config.modifier.urlRewrite.Rewrite(request.URL.Path)

	// Change HOST of original request
	URL := o.address + request.URL.Path + "?" + request.URL.RawQuery

	request.RequestURI = ""
	request.URL, _ = url.ParseRequestURI(URL)

	for _, header := range o.config.modifier.headers {
		SetHeader(request, header.Name, header.Value)
	}

	start := time.Now()
	resp, err := client.Do(request)
	stop := time.Now()

	// We should not count Redirect as errors
	if urlErr, ok := err.(*url.Error); ok {
		if _, ok := urlErr.Err.(*RedirectNotAllowed); ok {
			err = nil
		}
	}

	if err == nil {
		defer resp.Body.Close()
	} else {
		log.Println("Request error:", err)
	}

	if o.elasticSearch != nil {
		o.elasticSearch.ResponseAnalyze(request, resp, start, stop)
	}
}

func SetHeader(request *http.Request, name string, value string) {
	// Need to check here for the Host header as it needs to be set on the request and not as a separate header
	// http.ReadRequest sets it by default to the URL Host of the request being read
	if name == "Host" {
		request.Host = value
	} else {
		request.Header.Set(name, value)
	}

	return

}

func (o *HTTPOutput) String() string {
	return "HTTP output: " + o.address
}
