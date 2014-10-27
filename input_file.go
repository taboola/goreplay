package main

import (
	"encoding/gob"
	"log"
	"os"
	"time"
)

type FileInput struct {
	data        chan []byte
	path        string
	decoder     *gob.Decoder
	speedFactor float64
}

func NewFileInput(path string) (i *FileInput) {
	i = new(FileInput)
	i.data = make(chan []byte)
	i.path = path
	i.speedFactor = 1
	i.Init(path)

	go i.emit()

	return
}

func (i *FileInput) Init(path string) {
	file, err := os.Open(path)

	if err != nil {
		log.Fatal(i, "Cannot open file %q. Error: %s", path, err)
	}

	i.decoder = gob.NewDecoder(file)
}

func (i *FileInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	return len(buf), nil
}

func (i *FileInput) String() string {
	return "File input: " + i.path
}

func (i *FileInput) emit() {
	var lastTime int64

	for {
		raw := new(RawRequest)
		err := i.decoder.Decode(raw)

		if err != nil {
			return
		}

		if lastTime != 0 {
			timeDiff := raw.Timestamp - lastTime

			// We can speedup or slowdown execution based on speedFactor
			if i.speedFactor != 1 {
				timeDiff = int64(float64(raw.Timestamp-lastTime) / i.speedFactor)
			}

			time.Sleep(time.Duration(timeDiff))
		}

		lastTime = raw.Timestamp

		i.data <- raw.Request
	}
}
