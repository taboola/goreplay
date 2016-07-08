package main

import (
	"bytes"
	"io"
	"time"
)

// Start initialize loop for sending data from inputs to outputs
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
			finalize()
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// CopyMulty copies from 1 reader to multiple writers
func CopyMulty(src io.Reader, writers ...io.Writer) (err error) {
	buf := make([]byte, 5*1024*1024)
	wIndex := 0
	modifier := NewHTTPModifier(&Settings.modifierConfig)

	for {
		nr, er := src.Read(buf)

		if nr > 0 && len(buf) > nr {
			payload := buf[:nr]

			_maxN := nr
			if nr > 500 {
				_maxN = 500
			}

			if Settings.debug {
				Debug("[EMITTER] input:", string(payload[0:_maxN]), nr, "from:", src)
			}

			if modifier != nil && isRequestPayload(payload) {
				headSize := bytes.IndexByte(payload, '\n') + 1
				body := payload[headSize:]
				originalBodyLen := len(body)
				body = modifier.Rewrite(body)

				// If modifier tells to skip request
				if len(body) == 0 {
					continue
				}

				if originalBodyLen != len(body) {
					payload = append(payload[:headSize], body...)
				}

				if Settings.debug {
					Debug("[EMITTER] Rewrittern input:", len(payload), "First 500 bytes:", string(payload[0:_maxN]))
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
