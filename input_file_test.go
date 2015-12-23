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

func TestInputFileWithGET(t *testing.T) {

	input := NewTestInput()
	rg := NewRequestGenerator([]io.Reader{input}, func() { input.EmitGET() }, 1)
	readPayloads := [][]byte{}

	// Given a capture file with a GET request
	expectedCaptureFile := CreateCaptureFile(rg)
	defer expectedCaptureFile.TearDown()

	// When the request is read from the capture file
	err := ReadFromCaptureFile(expectedCaptureFile.file, 1, func(data []byte) {
		readPayloads = append(readPayloads, Duplicate(data))
	})

	// The read request should match the original request
	if err != nil {
		t.Error(err)
	} else {
		if !expectedCaptureFile.PayloadsEqual(readPayloads) {
			t.Error("Request read back from file should match")
		}
	}

}

func TestInputFileWithPayloadLargerThan64Kb(t *testing.T) {

	input := NewTestInput()
	rg := NewRequestGenerator([]io.Reader{input}, func() { input.EmitSizedPOST(64 * 1024) }, 1)
	readPayloads := [][]byte{}

	// Given a capture file with a request over 64Kb
	expectedCaptureFile := CreateCaptureFile(rg)
	defer expectedCaptureFile.TearDown()

	// When the request is read from the capture file
	err := ReadFromCaptureFile(expectedCaptureFile.file, 1, func(data []byte) {
		readPayloads = append(readPayloads, Duplicate(data))
	})

	// The read request should match the original request
	if err != nil {
		t.Error(err)
	} else {
		if !expectedCaptureFile.PayloadsEqual(readPayloads) {
			t.Error("Request read back from file should match")
		}
	}

}

func TestInputFileWithGETAndPOST(t *testing.T) {

	input := NewTestInput()
	rg := NewRequestGenerator([]io.Reader{input}, func() {
		input.EmitGET()
		input.EmitPOST()
	}, 2)
	readPayloads := [][]byte{}

	// Given a capture file with a GET request
	expectedCaptureFile := CreateCaptureFile(rg)
	defer expectedCaptureFile.TearDown()

	// When the requests are read from the capture file
	err := ReadFromCaptureFile(expectedCaptureFile.file, 2, func(data []byte) {
		readPayloads = append(readPayloads, Duplicate(data))
	})

	// The read requests should match the original request
	if err != nil {
		t.Error(err)
	} else {
		if !expectedCaptureFile.PayloadsEqual(readPayloads) {
			t.Error("Request read back from file should match")
		}
	}

}

type CaptureFile struct {
	data [][]byte
	file *os.File
}

func NewExpectedCaptureFile(data [][]byte, file *os.File) *CaptureFile {
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

type RequestGenerator struct {
	inputs []io.Reader
	emit   func()
	wg     *sync.WaitGroup
}

func NewRequestGenerator(inputs []io.Reader, emit func(), count int) (rg *RequestGenerator) {
	rg = new(RequestGenerator)
	rg.inputs = inputs
	rg.emit = emit
	rg.wg = new(sync.WaitGroup)
	rg.wg.Add(count)
	return
}

func (expectedCaptureFile *CaptureFile) PayloadsEqual(other [][]byte) bool {

	if len(expectedCaptureFile.data) != len(other) {
		return false
	}

	for i, payload := range other {

		if !bytes.Equal(expectedCaptureFile.data[i], payload) {
			return false
		}

	}

	return true

}

func CreateCaptureFile(requestGenerator *RequestGenerator) *CaptureFile {

	f, err := ioutil.TempFile("", "testmainconf")
	if err != nil {
		panic(err)
	}

	quit := make(chan int)

	readPayloads := [][]byte{}
	output := NewTestOutput(func(data []byte) {

		readPayloads = append(readPayloads, Duplicate(data))

		requestGenerator.wg.Done()
	})

	output_file := NewFileOutput(f.Name())

	Plugins.Inputs = requestGenerator.inputs
	Plugins.Outputs = []io.Writer{output, output_file}

	go Start(quit)

	requestGenerator.emit()
	requestGenerator.wg.Wait()

	close(quit)

	return NewExpectedCaptureFile(readPayloads, f)

}

func ReadFromCaptureFile(captureFile *os.File, count int, callback writeCallback) (err error) {

	quit := make(chan int)
	wg := new(sync.WaitGroup)

	input := NewFileInput(captureFile.Name())
	output := NewTestOutput(func(data []byte) {
		callback(data)
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	wg.Add(count)
	go Start(quit)

	done := make(chan int, 1)
	go func() {
		wg.Wait()
		done <- 1
	}()

	select {
	case <-done:
		break
	case <-time.After(2 * time.Second):
		err = errors.New("Timed out")
	}
	close(quit)

	return

}

func Duplicate(data []byte) (duplicate []byte) {
	duplicate = make([]byte, len(data))
	copy(duplicate, data)

	return
}
