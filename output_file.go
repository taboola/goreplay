package main

import (
	"io"
	"log"
	"os"
)

// FileOutput output plugin
type FileOutput struct {
	path    string
	file    *os.File
}

// NewFileOutput constructor for FileOutput, accepts path
func NewFileOutput(path string) io.Writer {
	o := new(FileOutput)
	o.path = path
	o.init(path)

	return o
}

func (o *FileOutput) init(path string) {
	var err error

	o.file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)

	if err != nil {
		log.Fatal(o, "Cannot open file %q. Error: %s", path, err)
	}
}

func (o *FileOutput) Write(data []byte) (n int, err error) {
	if !isOriginPayload(data) {
		return len(data), nil
	}

	o.file.Write(data)
	o.file.Write([]byte(payloadSeparator))

	return len(data), nil
}

func (o *FileOutput) String() string {
	return "File output: " + o.path
}
