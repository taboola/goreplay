package replay

import (
	"bytes"
	"encoding/gob"
	"io"
	"io/ioutil"
	"log"

	"fmt"
)

type ParsedRequest struct {
	Timestamp int64
	Request   []byte
}

func (self ParsedRequest) String() string {
	return fmt.Sprintf("Request: %v, timestamp: %v", string(self.Request), self.Timestamp)
}

func parseReplayFile() (requests []ParsedRequest, err error) {
	requests, err = readLines(Settings.FileToReplayPath)

	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	return
}

// readLines reads a whole file into memory
// and returns a slice of request+timestamps.
func readLines(path string) (requests []ParsedRequest, err error) {
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	fileBuf := bytes.NewBuffer(file)
	fileDec := gob.NewDecoder(fileBuf)

	for err == nil {
		var reqBuf ParsedRequest
		err = fileDec.Decode(&reqBuf)

		if err == io.EOF {
			err = nil
			break
		}

		requests = append(requests, reqBuf)
	}

	return requests, err
}
