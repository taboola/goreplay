package main

import (
	"testing"

	"github.com/buger/gor/listener"
	"github.com/buger/gor/replay"

	"time"

	"fmt"
	"net/http"
	"strconv"
)

func isEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Error("Original and Replayed request not match\n", a, "!=", b)
	}
}

func startListener() {
	listener.Settings.Verbose = true
	listener.Settings.Address = "127.0.0.1"
	listener.Settings.ReplayAddress = "127.0.0.1:50001"
	listener.Settings.Port = 50000
	listener.Run()
}

func startReplay() {
	replay.Settings.Verbose = true
	replay.Settings.Host = "127.0.0.1"
	replay.Settings.ForwardAddress = "127.0.0.1:50002"
	replay.Settings.Port = 50001
	replay.Run()
}

func startHTTP(port int, handler http.Handler) {
	http.ListenAndServe(":"+strconv.Itoa(port), handler)
}

func getRequest() *http.Request {
	req, _ := http.NewRequest("GET", "http://localhost:50000/test", nil)
	ck1 := new(http.Cookie)
	ck1.Name = "test"
	ck1.Value = "value"

	req.AddCookie(ck1)

	return req
}

func startEnv(listenHandler http.HandlerFunc, replayHandler http.HandlerFunc) {
	go startHTTP(50000, http.HandlerFunc(listenHandler))
	go startListener()
	go startReplay()
	go startHTTP(50002, http.HandlerFunc(replayHandler))
}

func TestReplay(t *testing.T) {
	request := getRequest()
	received := make(chan int)

	listenHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "404 page not found", http.StatusNotFound)
	}

	replayHandler := func(w http.ResponseWriter, r *http.Request) {
		isEqual(t, r.URL.Path, request.URL.Path)
		isEqual(t, r.Cookies()[0].Value, request.Cookies()[0].Value)

		http.Error(w, "404 page not found", http.StatusNotFound)

		if t.Failed() {
			fmt.Println("\nReplayed:", r, "\nOriginal:", request)
		}

		received <- 1
	}

	startEnv(listenHandler, replayHandler)

	// Time to start http and gor instances
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
}

func TestRateLimit(t *testing.T) {

}
