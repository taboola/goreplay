package main

import (
	"io"
	"time"
	"crypto/rand"
)

func uuid() []byte {
        b := make([]byte, 16)
        rand.Read(b)
        return b
}


func Start(stop chan int) {
	if Settings.middleware != "" {
		middleware := NewMiddleware(Settings.middleware)

		for _, in := range Plugins.Inputs {
			middleware.ReadFrom(in)
		}

		// We going only to read responses, so using same ReadFrom method
		for _, out := range Plugins.Outputs {
			if r, ok := out.(io.Reader); ok {
				middleware.ReadFrom(r)
			}
		}

		go CopyMulty(middleware, Plugins.Outputs...)
	} else {
		for _, in := range Plugins.Inputs {
			go CopyMulty(in, Plugins.Outputs...)
		}
	}

	for {
		select {
		case <-stop:
			return
		case <-time.After(1 * time.Second):
		}
	}
}

// Copy from 1 reader to multiple writers
func CopyMulty(src io.Reader, writers ...io.Writer) (err error) {
	buf := make([]byte, 5*1024*1024)
	wIndex := 0
	modifier := NewHTTPModifier(&Settings.modifierConfig)

	for {
		nr, er := src.Read(buf)

		if nr > 0 && len(buf) > nr {
			payload := buf[0:nr]

			Debug("[EMITTER] input:", string(payload))

			if modifier != nil {
				payload = modifier.Rewrite(payload)

				// If modifier tells to skip request
				if len(payload) == 0 {
					continue
				}
			}

			if Settings.splitOutput {
				// Simple round robin
				writers[wIndex].Write(payload)

				wIndex++

				if wIndex >= len(writers) {
					wIndex = 0
				}
			} else {
				for _, dst := range writers {
					dst.Write(payload)
				}
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
