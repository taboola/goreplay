package listener

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func getTCPMessage() (msg *TCPMessage) {
	packet := &TCPPacket{Data: []byte("GET /pub/WWW/ HTTP/1.1\nHost: www.w3.org\r\n\r\n")}

	return &TCPMessage{packets: []*TCPPacket{packet}}
}

func mockReplayServer() (listener net.Listener) {
	listener, _ = net.Listen("tcp", "127.0.0.1:0")

	Settings.ReplayAddress = listener.Addr().String()

	fmt.Println(listener.Addr().String())

	return
}

func TestSendMessage(t *testing.T) {
	Settings.Verbose = true

	listener := mockReplayServer()

	msg := getTCPMessage()

	sendMessage(msg)

	conn, _ := listener.Accept()
	defer conn.Close()

	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	buf = buf[0:n]

	if bytes.Compare(buf, msg.Bytes()) != 0 {
		t.Errorf("Original and reveived requests does not match")
	}
}

func TestSaveMessageToFile(t *testing.T) {
	Settings.Verbose = true
	Settings.FileToReplyPath = "requests.gor"
	Settings.Address = "127.0.0.1"
	Settings.Port = 50000

	received := make(chan int)

	requestBytes := []byte("GET / HTTP/1.1\nHost: localhost:50000\r\n\r\n")

	handler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "OK", http.StatusNotFound)
		received <- 1
	}

	go func() {
		http.ListenAndServe(":50000", http.HandlerFunc(handler))
	}()

	time.Sleep(time.Millisecond * 100)
	time.Sleep(time.Millisecond * 100)
	time.Sleep(time.Millisecond * 100)
	go Run()
	go func() {
		conn, _ := net.Dial("tcp", ":50000")
		conn.Write(requestBytes)
	}()

	select {
	case <-received:
		time.Sleep(time.Millisecond * 100)
	case <-time.After(time.Second):
		t.Error("Timeout error")
	}

	file, err := os.Open("requests.gor")

	if err != nil {
		t.Errorf("Problem with opening file: ", err)
	}

	fileBuf := make([]byte, 100)
	file.Read(fileBuf)

	if bytes.Compare(fileBuf, requestBytes) != 0 {
		t.Errorf("Original and received requests does not match")
	}
}
