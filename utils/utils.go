package utils

import (
	"fmt"
)

type ParsedRequest struct {
	Timestamp int64
	Request   []byte
}

func (self ParsedRequest) String() string {
	return fmt.Sprintf("Request: %v, timestamp: %v", string(self.Request), self.Timestamp)
}
