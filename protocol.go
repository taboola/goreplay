package main

import (
	"strconv"
)

const (
	RequestPayload = 1 << iota
	ResponsePayload
	ReplayedResponsePayload
)

// Timing is request start or round-trip time, depending on payloadType
func payloadHeader(payloadType int, uuid []byte, timing int64) (header []byte) {
	sTime := strconv.FormatInt(timing, 10)

	//Example:
	//  3 f45590522cd1838b4a0d5c5aab80b77929dea3b3 1231\n
	// `+ 1` indicates space characters or end of line
	header = make([]byte, 1+1+len(uuid)+1+len(sTime)+1)
	header[1] = ' '
	header[2+len(uuid)] = ' '
	header[len(header)-1] = '\n'

	switch payloadType {
	case RequestPayload:
		header[0] = '1'
	case ResponsePayload:
		header[0] = '2'
	case ReplayedResponsePayload:
		header[0] = '3'
	}

	copy(header[2:], uuid)
	copy(header[3+len(uuid):], sTime)

	return header
}
