package rawSocket

import (
    "testing"
    "time"
    "bytes"
    _ "log"
)

func TestRawListenerInput(t *testing.T) {
    var req, resp *TCPMessage

    listener := NewListener("", "0", 10 * time.Millisecond)
    defer listener.Close()

    reqPacket := buildPacket(true, 1, 1, []byte("GET / HTTP/1.1"))

    listener.packetsChan <- reqPacket

    respAck := reqPacket.Seq + uint32(len(reqPacket.Data))
    respPacket := buildPacket(false, respAck, reqPacket.Seq + 1, []byte("HTTP/1.1 200 OK"))
    listener.packetsChan <- respPacket


    select {
        case req = <- listener.messagesChan:
        case <- time.After(time.Millisecond):
            t.Error("Should return respose immediately")
            return
    }

    if !req.IsIncoming {
        t.Error("Should be request")
    }

    select {
        case resp = <- listener.messagesChan:
        case <- time.After(time.Millisecond):
            t.Error("Should return response immediately")
            return
    }

    if resp.IsIncoming {
        t.Error("Should be response")
    }
}

func TestRawListenerResponse(t *testing.T) {
    var req, resp *TCPMessage

    listener := NewListener("", "0", 10 * time.Millisecond)
    defer listener.Close()

    reqPacket := buildPacket(true, 1, 1, []byte("GET / HTTP/1.1"))
    respPacket := buildPacket(false, 1 + uint32(len(reqPacket.Data)), 2, []byte("HTTP/1.1 200 OK"))

    // If response packet comes before request
    listener.packetsChan <- respPacket
    listener.packetsChan <- reqPacket

    select {
        case req = <- listener.messagesChan:
        case <- time.After(time.Millisecond):
            t.Error("Should return respose immediately")
            return
    }

    if !req.IsIncoming {
        t.Error("Should be request")
    }

    select {
        case resp = <- listener.messagesChan:
        case <- time.After(time.Millisecond):
            t.Error("Should return response immediately")
            return
    }

    if resp.IsIncoming {
        t.Error("Should be response")
    }

    if !bytes.Equal(resp.UUID(), req.UUID()) {
        t.Error("Resp and Req UUID should be equal")
    }
}

func TestRawListener100Continue(t *testing.T) {
    var req, resp *TCPMessage

    listener := NewListener("", "0", 10 * time.Millisecond)
    defer listener.Close()

    reqPacket1 := buildPacket(true, 1, 1, []byte("POST / HTTP/1.1\r\nContent-Length: 2\r\nExpect: 100-continue\r\n\r\n"))
    // Packet with data have different Seq
    reqPacket2 := buildPacket(true, 2, reqPacket1.Seq + uint32(len(reqPacket1.Data)), []byte("a"))
    reqPacket3 := buildPacket(true, 2, reqPacket2.Seq + 1, []byte("b"))

    respPacket1 := buildPacket(false, 10, 3, []byte("HTTP/1.1 100 Continue\r\n"))
    respPacket2 :=  buildPacket(false, reqPacket3.Seq + uint32(len(reqPacket1.Data)) + uint32(len(reqPacket2.Data)) + uint32(len(reqPacket3.Data)), 2, []byte("HTTP/1.1 200 OK\r\n"))

    listener.processTCPPacket(reqPacket1)
    listener.processTCPPacket(reqPacket2)
    listener.processTCPPacket(reqPacket3)

    listener.processTCPPacket(respPacket1)
    listener.processTCPPacket(respPacket2)

    select {
        case req = <- listener.messagesChan:
            break
        case <- time.After(11 * time.Millisecond):
            t.Error("Should return respose after expire time")
            return
    }

    if !bytes.Equal(req.Bytes(), []byte("POST / HTTP/1.1\r\nContent-Length: 2\r\n\r\nab")) {
        t.Error("Should recived full message")
    }

    if !req.IsIncoming {
        t.Error("Should be request")
    }

    select {
        case resp = <- listener.messagesChan:
            break
        case <- time.After(21 * time.Millisecond):
            t.Error("Should return response after expire time")
            return
    }

    if resp.IsIncoming {
        t.Error("Should be response")
    }

    if !bytes.Equal(resp.UUID(), req.UUID()) {
        t.Error("Resp and Req UUID should be equal")
    }
}
