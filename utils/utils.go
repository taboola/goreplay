package utils

import (
	"fmt"
)

type RawRequest struct {
	Timestamp int64
	Request   []byte
}

func (self RawRequest) String() string {
	return fmt.Sprintf("Request: %v, timestamp: %v", string(self.Request), self.Timestamp)
}
