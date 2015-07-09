// Low-level interaction with HTTP request payload
package proto

import (
    "bytes"
    "github.com/buger/gor/byteutils"
    _ "log"
)

var CLRF = []byte("\r\n")
var EMPTY_LINE = []byte("\r\n\r\n")
var HEADER_DELIM = []byte(": ")

// Headers should end with empty line
func MIMEHeadersEndPos(payload []byte) int {
    return bytes.Index(payload, EMPTY_LINE)
}

func MIMEHeadersStartPos(payload []byte) int {
    return bytes.Index(payload, CLRF) + 2 // Find first line end
}

// Find header value or return error
// Do not support multi-line headers
func Header(payload []byte, name []byte) (value []byte, headerStart, valueStart, headerEnd int) {
    headerStart = bytes.Index(payload, name)

    if headerStart == -1 {
        return
    }

    valueStart = headerStart + len(name) + 1 // Skip ":" after header name
    if payload[valueStart] == ' ' { // Ignore empty space after ':'
        valueStart += 1
    }
    headerEnd = valueStart + bytes.IndexByte(payload[valueStart:], '\r')
    value = payload[valueStart:headerEnd]

    return
}

func GetHeader(payload []byte, name string) []byte {
    val, _, _, _ := Header(payload, []byte(name))

    return val
}

func SetHeader(payload, name, value []byte) []byte {
    _, hs, vs, he := Header(payload, name)

    // If header found
    if hs != -1 {
        return byteutils.Replace(payload, vs, he, value)
    } else {
        return AddHeader(payload, name, value)
    }
}

func AddHeader(payload, name, value []byte) []byte {
    header := make([]byte, len(name) + 2 + len(value) + 2)
    copy(header[0:], name)
    copy(header[len(name):], HEADER_DELIM)
    copy(header[len(name)+2:], value)
    copy(header[len(header)-2:], CLRF)

    mimeStart := MIMEHeadersStartPos(payload)

    return byteutils.Insert(payload, mimeStart, header)
}

func Path(payload []byte) []byte {
    start := bytes.IndexByte(payload, ' ')
    start += 1

    end := bytes.IndexByte(payload[start:], ' ')

    return payload[start:start+end]
}

func SetPath(payload, path []byte) []byte {
    start := bytes.IndexByte(payload, ' ')
    start += 1

    end := bytes.IndexByte(payload[start:], ' ')

    return byteutils.Replace(payload, start, start+end, path)
}

func PathParam(payload, name []byte) (value []byte, valueStart, valueEnd int) {
    path := Path(payload)

    if paramStart := bytes.Index(path, append(name, '=')); paramStart != -1 {
        valueStart := paramStart + len(name) + 1
        paramEnd := bytes.IndexByte(path[valueStart:], '&')
        if paramEnd == -1 { // It is final param
            paramEnd = len(path)
        } else {
            paramEnd += valueStart
        }

        return path[valueStart:paramEnd], valueStart, paramEnd
    } else {
        return []byte(""), -1, -1
    }
}

func SetHost(payload, url, host []byte) []byte {
    // If this is HTTP 1.0 traffic or proxy traffic it may include host right into path variable, so instead of setting Host header we rewrite Path
    // Fix for https://github.com/buger/gor/issues/156
    if path := Path(payload); bytes.HasPrefix(path, []byte("http")) {
        hostStart := bytes.IndexByte(path, ':') // : position "https?:"
        hostStart += 3 // Skip 1 ':' and 2 '\'
        hostEnd := hostStart + bytes.IndexByte(path[hostStart:], '/')

        newPath := make([]byte, len(path))
        copy(newPath, path)
        newPath = byteutils.Replace(newPath, 0, hostEnd, url)

        return SetPath(payload, newPath)
    } else {
        return SetHeader(payload, []byte("Host"), host)
    }
}

func Method(payload []byte) []byte {
    end := bytes.IndexByte(payload, ' ')

    return payload[:end]
}

// Status in response have same position as Path in request
func Status(payload []byte) []byte {
    return Path(payload)
}