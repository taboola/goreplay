package listener

import (
  "log"
	"os"
)

type MessageLogger struct {
  *log.Logger

	file *os.File
}

func NewLog(filename string) *MessageLogger {

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)

	if err != nil {
		log.Fatal("Cannot open file %q. Error: %s", filename, err)
	}

  logger := log.New(file, "", 0)

	messageLogger := &MessageLogger{
    Logger: logger,
	}

	return messageLogger
}

func (messageLogger *MessageLogger) close() {
	messageLogger.file.Close()
}
