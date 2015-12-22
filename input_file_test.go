package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestInputFileWithPayloadLargerThan64Kb(t *testing.T) {

	// Given a capture file with a request over 64Kb
	expectedCaptureFile := CreateCaptureFileWithOneRequest()
	defer expectedCaptureFile.TearDown()

	// When the request is read from the capture file
	readCapture, err := ReadFromCaptureFile(expectedCaptureFile.file)

	// The read request should match the original request
	if err != nil {
		t.Error(err)
	} else {
		if !expectedCaptureFile.DataEquals(readCapture) {
			t.Error("Request read back from file should match")
		}
	}

}

type CaptureFile struct {
	data []byte
	file *os.File
}

func NewExpectedCaptureFile(data []byte, file *os.File) *CaptureFile {
	ecf := new(CaptureFile)
	ecf.file = file
	ecf.data = data
	return ecf
}

func (expectedCaptureFile *CaptureFile) TearDown() {
	if expectedCaptureFile.file != nil {
		syscall.Unlink(expectedCaptureFile.file.Name())
	}
}

func (expectedCaptureFile *CaptureFile) DataEquals(other []byte) bool {
	return bytes.Equal(expectedCaptureFile.data, other)
}

func CreateCaptureFileWithOneRequest() *CaptureFile {

	f, err := ioutil.TempFile("", "testmainconf")
	if err != nil {
		panic(err)
	}

	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewTestInput()

	var buffer bytes.Buffer
	output := NewTestOutput(func(data []byte) {
		buffer.Write(data)
		wg.Done()
	})

	output_file := NewFileOutput(f.Name())

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output, output_file}

	wg.Add(1)
	go Start(quit)

	input.EmitSizedPOST(64 * 1024)
	wg.Wait()

	close(quit)

	return NewExpectedCaptureFile(buffer.Bytes(), f)

}

func ReadFromCaptureFile(captureFile *os.File) (read []byte, err error) {

	quit := make(chan int)
	wg := new(sync.WaitGroup)

	var buffer2 bytes.Buffer

	input := NewFileInput(captureFile.Name())
	output := NewTestOutput(func(data []byte) {
		buffer2.Write(data)
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	wg.Add(1)
	go Start(quit)

	done := make(chan int, 1)
	go func() {
		wg.Wait()
		done <- 1
	}()

	select {
	case <-done:
		read = buffer2.Bytes()
	case <-time.After(2 * time.Second):
		err = errors.New("Timed out")
	}
	close(quit)

	return

}
