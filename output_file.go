package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var dateFileNameFuncs = map[string]func() string{
	"%Y":  func() string { return time.Now().Format("2006") },
	"%m":  func() string { return time.Now().Format("01") },
	"%d":  func() string { return time.Now().Format("02") },
	"%H":  func() string { return time.Now().Format("15") },
	"%M":  func() string { return time.Now().Format("04") },
	"%S":  func() string { return time.Now().Format("05") },
	"%NS": func() string { return fmt.Sprint(time.Now().Nanosecond()) },
}

type FileOutputConfig struct {
	flushInterval time.Duration
	sizeLimit     unitSizeVar
	queueLimit    int
	append        bool
}

// FileOutput output plugin
type FileOutput struct {
	pathTemplate string
	currentName  string
	file         *os.File
	queueLength  int
	chunkSize    int
	writer       io.Writer

	config *FileOutputConfig
}

// NewFileOutput constructor for FileOutput, accepts path
func NewFileOutput(pathTemplate string, config *FileOutputConfig) *FileOutput {
	o := new(FileOutput)
	o.pathTemplate = pathTemplate
	o.config = config
	o.updateName()

	// Force flushing every minute
	go func() {
		for {
			time.Sleep(o.config.flushInterval)
			o.flush()
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second)
			o.updateName()
		}
	}()

	return o
}

func getFileIndex(name string) int {
	ext := filepath.Ext(name)
	withoutExt := strings.TrimSuffix(name, ext)

	if idx := strings.LastIndex(withoutExt, "_"); idx != -1 {
		if i, err := strconv.Atoi(withoutExt[idx+1:]); err == nil {
			return i
		}
	}

	return -1
}

func setFileIndex(name string, idx int) string {
	idxS := strconv.Itoa(idx)
	ext := filepath.Ext(name)
	withoutExt := strings.TrimSuffix(name, ext)

	if i := strings.LastIndex(withoutExt, "_"); i != -1 {
		withoutExt = withoutExt[:i]
	}

	return withoutExt + "_" + idxS + ext
}

func (o *FileOutput) filename() string {
	path := o.pathTemplate

	for name, fn := range dateFileNameFuncs {
		path = strings.Replace(path, name, fn(), -1)
	}

	if !o.config.append {
		nextChunk := false

		if o.currentName == "" ||
			((o.config.queueLimit > 0 && o.queueLength >= o.config.queueLimit) ||
				(o.config.sizeLimit > 0 && o.chunkSize >= int(o.config.sizeLimit))) {
			nextChunk = true
		}

		ext := filepath.Ext(path)
		withoutExt := strings.TrimSuffix(path, ext)

		if matches, err := filepath.Glob(withoutExt + "*" + ext); err == nil {
			if len(matches) == 0 {
				return setFileIndex(path, 0)
			}
			sort.Strings(matches)

			last := matches[len(matches)-1]

			if idx := getFileIndex(last); idx != -1 {
				if nextChunk {
					return setFileIndex(last, idx+1)
				} else {
					return setFileIndex(last, idx)
				}
			} else {
				return setFileIndex(last, 0)
			}
		}
	}

	return path
}

func (o *FileOutput) updateName() {
	o.currentName = o.filename()
}

func (o *FileOutput) Write(data []byte) (n int, err error) {
	if !isOriginPayload(data) {
		return len(data), nil
	}

	if o.file == nil || o.currentName != o.file.Name() {
		o.Close()

		o.file, err = os.OpenFile(o.currentName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
		o.file.Sync()

		if strings.HasSuffix(o.currentName, ".gz") {
			o.writer = gzip.NewWriter(o.file)
		} else {
			o.writer = bufio.NewWriter(o.file)
		}

		if err != nil {
			log.Fatal(o, "Cannot open file %q. Error: %s", o.currentName, err)
		}

		o.queueLength = 0
	}

	o.writer.Write(data)
	o.writer.Write([]byte(payloadSeparator))

	o.queueLength++

	return len(data), nil
}

func (o *FileOutput) flush() {
	if o.file != nil {
		if strings.HasSuffix(o.currentName, ".gz") {
			o.writer.(*gzip.Writer).Flush()
		} else {
			o.writer.(*bufio.Writer).Flush()
		}
	}

	if stat, err := o.file.Stat(); err != nil {
		o.chunkSize = int(stat.Size())
	}
}

func (o *FileOutput) String() string {
	return "File output: " + o.file.Name()
}

func (o *FileOutput) Close() {
	if o.file != nil {
		if strings.HasSuffix(o.currentName, ".gz") {
			o.writer.(*gzip.Writer).Close()
		} else {
			o.writer.(*bufio.Writer).Flush()
		}
		o.file.Close()
	}
}
