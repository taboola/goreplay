package main

const (
	RequestPayload = 1 << iota
	ResponsePayload
	ReplayedResponsePayload
)

func payloadHeader(payloadType int, uuid []byte) (header []byte) {
	header = make([]byte, 43)
	header[1] = ' '
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

	return header
}
