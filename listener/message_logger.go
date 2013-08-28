package listener

import (
	"fmt"
	"os"
)

type MessageLogger struct {
	messageChannel chan string

	file *os.File
}

func NewLog(filename string) *MessageLogger {

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)

	if err != nil {
		panic(fmt.Sprintf("Cannot open file %q. Error: %s", filename, err))
	}

	messageLogger := &MessageLogger{
		messageChannel: make(chan string),
		file:           file,
	}

	go func() {
		defer func() {
			messageLogger.close()
		}()

		for {
			select {
			case message := <-messageLogger.messageChannel:
				messageLogger.log(message)
			}
		}
	}()

	return messageLogger
}

func (messageLogger *MessageLogger) log(message string) {
	fmt.Fprintln(messageLogger.file, message)
}

func (messageLogger *MessageLogger) close() {
	messageLogger.file.Close()
}
