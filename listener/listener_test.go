package listener

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
  "time"
  "io"
)

func getTCPMessage() (msg *TCPMessage) {
	packet := &TCPPacket{Data: []byte("GET /pub/WWW/ HTTP/1.1\nHost: www.w3.org\r\n\r\n")}

	return &TCPMessage{packets: []*TCPPacket{packet}}
}

func mockReplayServer() (listener net.Listener) {
	listener, _ = net.Listen("tcp", "127.0.0.1:0")

	Settings.ReplayAddress = listener.Addr().String()

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
	Settings.FileToReplyPath = "listener_test.gor"
	Settings.Address = "127.0.0.1"
	Settings.Port = 50000

	receivedChan := make(chan int)

  // receivedChan <- 1
	// requestBytes := []byte("GET / HTTP/1.1\nHost: localhost:50000\r\n\r\n")

	handler := func(w http.ResponseWriter, r *http.Request) {
    fmt.Println("handler called")
    // fmt.Fprintf(w, "Hello, aaa")
    // this is faulty
    io.WriteString(w, "hello, world!\n")
	}

	go Run()

	go func() {
		http.ListenAndServe(":50000", http.HandlerFunc(handler))
	}()


	select {
  case msg, ok := <-receivedChan:
    fmt.Println("received something")
	case <-time.After(time.Second):
    fmt.Println("in timeout section")
    // t.Error("Server not started and I dont know what is going on :(")
	}

  request := getRequest()
  resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("Problem with default client", err)
	}
  fmt.Println("RESPONSE", resp)

	file, err := os.Open("listener_test.gor")

	if err != nil {
		t.Errorf("Problem with opening file: ", err)
	}

	fileBuf := make([]byte, 1024)
  n, err  := file.Read(fileBuf)
  fileBuf = fileBuf[:n]

	//requestBuffer := bytes.NewBuffer(fileBuf)
  //requestReader := bufio.NewReader(requestBuffer)
  //readRequest, _ := http.ReadRequest(requestReader)
  fmt.Println("Read file: \n", string(fileBuf))
  fmt.Println("Read file: \n", fileBuf)

	//if bytes.Compare(fileBuf, make([]byte, 1024)) != 0 {
  //		t.Errorf("Original and received requests does not match")
	//}
  // if *request != *readRequest {
  	// t.Errorf("Original and received requests does not match")
  //}
  t.Errorf("Original and received requests does not match")
}

func getRequest() (req *http.Request) {
	req, _ = http.NewRequest("GET", "http://localhost:50000", nil)
	ck := new(http.Cookie)
	ck.Name = "test"
	ck.Value = "value2"

	req.AddCookie(ck)

	return
}
