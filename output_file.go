package gor

import (
	"log"
	"os"
	"time"
)

type FileOutput struct {
	path   string
	logger *log.Logger
}

func NewFileOutput(path string) (o *FileOutput) {
	o = new(FileOutput)
	o.path = path
	o.Init(path)

	return
}

func (o *FileOutput) Init(path string) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)

	if err != nil {
		log.Fatal(o, "Cannot open file %q. Error: %s", path, err)
	}

	o.logger = log.New(file, "", 0)
}

func (o *FileOutput) Write(data []byte) (n int, err error) {
	log.Printf("%v\n%s\n", time.Now().UnixNano(), string(data))
	o.logger.Printf("%v\n%s\n", time.Now().UnixNano(), string(data))

	return len(data), nil
}

func (o *FileOutput) String() string {
	return "File output: " + o.path
}
