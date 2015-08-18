package main

import (
	"strconv"
	"bytes"
	"encoding/hex"
	"crypto/rand"
)

const (
	RequestPayload = 1 << iota
	ResponsePayload
	ReplayedResponsePayload
)

func uuid() []byte {
	b := make([]byte, 20)
	rand.Read(b)

	uuid := make([]byte, 40)
	hex.Encode(uuid, b)

	return uuid
}


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

func payloadBody(payload []byte) []byte {
	headerSize := bytes.IndexByte(payload, '\n')
	return payload[headerSize+1:]
}

func payloadMeta(payload []byte) [][]byte {
	headerSize := bytes.IndexByte(payload, '\n')
	return bytes.Split(payload[:headerSize], []byte{' '})
}

func isOriginPayload(payload []byte) bool {
	switch payload[0] {
	case '1', '2':
		return true
	default:
		return false
	}
}

func isRequestPayload(payload []byte) bool {
	return payload[0] == '1'
}