package main

import (
	"testing"

	"github.com/buger/gor/listener"
	"github.com/buger/gor/replay"

	"time"

	"net/http"
	"strconv"
)

func startListener() {
	listener.Settings.Verbose = true
	listener.Settings.Address = "127.0.0.1"
	listener.Settings.ReplayAddress = "127.0.0.1:50001"
	listener.Settings.Port = 50000
	go listener.Run()
}

func startReplay() {
	replay.Settings.Verbose = true
	replay.Settings.Host = "127.0.0.1"
	replay.Settings.ForwardAddress = "127.0.0.1:50002"
	replay.Settings.Port = 50001
	go replay.Run()
}

func startHTTP(port int, handler http.Handler) {
	go http.ListenAndServe(":"+strconv.Itoa(port), handler)
}

func getRequest() *http.Request {
	req, _ := http.NewRequest("GET", "http://localhost:50000/test", nil)
	ck1 := new(http.Cookie)
	ck1.Name = "test"
	ck1.Value = "value"

	req.AddCookie(ck1)

	return req
}

func TestIntegration(t *testing.T) {
	request := getRequest()

	listenHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "404 page not found", http.StatusNotFound)
	}
	startHTTP(50000, http.HandlerFunc(listenHandler))

	startListener()
	startReplay()

	received := make(chan int)

	replayHandler := func(w http.ResponseWriter, r *http.Request) {
		equal := func(a interface{}, b interface{}) {
			if a != b {
				t.Error("Original and Replayed request not match\n", a, "!=", b, "\nReplayed:", r, "\nOriginal:", request)
			}
		}

		equal(r.URL.Path, request.URL.Path)
		equal(r.Cookies()[0].Value, request.Cookies()[0].Value)

		http.Error(w, "404 page not found", http.StatusNotFound)

		received <- 1
	}
	startHTTP(50002, http.HandlerFunc(replayHandler))

	time.Sleep(time.Millisecond * 100)

	_, err := http.DefaultClient.Do(request)

	if err != nil {
		t.Error("Can't make request", err)
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Error("Timeout error")
	}

	time.Sleep(time.Millisecond * 500)
}
