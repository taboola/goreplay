package main

import (
	"bytes"
	"fmt"
	"strings"
)

type HTTPMethods [][]byte

func (h *HTTPMethods) String() string {
	return fmt.Sprint(*h)
}

func (h *HTTPMethods) Set(value string) error {
	*h = append(*h, []byte(strings.ToUpper(value)))
	return nil
}

func (h *HTTPMethods) Contains(value []byte) bool {
	for _, method := range *h {
		if bytes.Equal(value, method) {
			return true
		}
	}
	return false
}
