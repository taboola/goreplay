package main

import (
	"io"
	"log"
	"sync/atomic"
	"time"
)

const InitialDynamicWorkers = 10

type HTTPOutputConfig struct {
	redirectLimit int

	stats   bool
	workers int

	elasticSearch string

	Debug bool
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
	client := NewHTTPClient(o.address, &HTTPClientConfig{
		FollowRedirects: o.config.redirectLimit,
		Debug:           o.config.Debug,
	})

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

func (o *HTTPOutput) sendRequest(client *HTTPClient, request []byte) {
	start := time.Now()
	resp, err := client.Send(request)
	stop := time.Now()

	if err != nil {
		log.Println("Request error:", err)
	}

	if o.elasticSearch != nil {
		o.elasticSearch.ResponseAnalyze(request, resp, start, stop)
	}
}

func (o *HTTPOutput) String() string {
	return "HTTP output: " + o.address
}
