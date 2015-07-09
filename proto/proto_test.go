package proto

import (
    "testing"
    "bytes"
)

func TestHeader(t *testing.T) {
    var payload, val []byte
    var headerStart int

    payload = []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if val, _, _, _ = Header(payload, []byte("Content-Length")); !bytes.Equal(val, []byte("7")) {
        t.Error("Should find header value")
    }

    payload = []byte("POST /post HTTP/1.1\r\nContent-Length:7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if val, _, _, _ = Header(payload, []byte("Content-Length")); !bytes.Equal(val, []byte("7")) {
        t.Error("Should find header value without space after :")
    }

    if _, headerStart, _, _ = Header(payload, []byte("Not-Found")); headerStart != -1  {
        t.Error("Should not found header")
    }
}

func TestMIMEHeadersEndPos(t *testing.T) {
    head := []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org")
    payload := []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    end := MIMEHeadersEndPos(payload)

    if !bytes.Equal(payload[:end], head) {
        t.Error("Wrong headers end position:", end)
    }
}

func TestMIMEHeadersStartPos(t *testing.T) {
    headers := []byte("Content-Length: 7\r\nHost: www.w3.org")
    payload := []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    start := MIMEHeadersStartPos(payload)
    end := MIMEHeadersEndPos(payload)

    if !bytes.Equal(payload[start:end], headers) {
        t.Error("Wrong headers end position:", start, end)
    }
}

func TestSetHeader(t *testing.T) {
    var payload, payload_after []byte

    payload = []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")
    payload_after = []byte("POST /post HTTP/1.1\r\nContent-Length: 14\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if payload = SetHeader(payload, []byte("Content-Length"), []byte("14")); !bytes.Equal(payload, payload_after) {
        t.Error("Should update header if it exists", string(payload))
    }


    payload = []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")
    payload_after = []byte("POST /post HTTP/1.1\r\nUser-Agent: Gor\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if payload = SetHeader(payload, []byte("User-Agent"), []byte("Gor")); !bytes.Equal(payload, payload_after) {
        t.Error("Should add header if not found", string(payload))
    }
}

func TestPath(t *testing.T) {
    var path, payload []byte

    payload = []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if path = Path(payload); !bytes.Equal(path, []byte("/post")) {
        t.Error("Should find path", string(path))
    }
}

func TestSetPath(t *testing.T) {
    var payload, payload_after []byte

    payload = []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")
    payload_after = []byte("POST /new_path HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if payload = SetPath(payload, []byte("/new_path")); !bytes.Equal(payload, payload_after) {
        t.Error("Should replace path", string(payload))
    }
}

func TestPathParam(t *testing.T) {
    var payload []byte

    payload = []byte("POST /post?param=test&user_id=1 HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if val, _, _ := PathParam(payload, []byte("param")); !bytes.Equal(val, []byte("test")) {
        t.Error("Should detect attribute", string(val))
    }

    if val, _, _ := PathParam(payload, []byte("user_id")); !bytes.Equal(val, []byte("1")) {
        t.Error("Should detect attribute", string(val))
    }
}


func TestSetHostHTTP10(t *testing.T) {
    var payload, payload_after []byte

    payload = []byte("POST http://example.com/post HTTP/1.0\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")
    payload_after = []byte("POST http://new.com/post HTTP/1.0\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if payload = SetHost(payload, []byte("http://new.com"), []byte("new.com")); !bytes.Equal(payload, payload_after) {
        t.Error("Should replace host", string(payload))
    }
}