package gor

import (
	"io"
	"time"
)

func Start() {
	for _, in := range Plugins.Inputs {
		CopyMulty(in, Plugins.Outputs...)
	}

	for {
		time.Sleep(time.Second)
	}
}

// Copy from 1 reader to multiple writers
func CopyMulty(src io.Reader, writers ...io.Writer) (err error) {
	buf := make([]byte, 32*1024)

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			for _, dst := range writers {
				dst.Write(buf[0:nr])
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return err
}
