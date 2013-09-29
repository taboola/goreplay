package replay

import (
  "bufio"
  "log"
  "os"
  "bytes"
)

func parseReplyFile() (requests [][]byte, err error) {
  requests, err = readLines(Settings.FileToReplyPath)

  if err != nil {
    log.Fatalf("readLines: %s", err)
  }

  return
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) (requests [][]byte, err error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  scanner := bufio.NewScanner(file)
  scanner.Split(scanLinesFunc)
  for scanner.Scan() {
    if len(scanner.Text()) > 5 {
      requests = append(requests, scanner.Bytes())
    }
  }
  return requests, scanner.Err()
}

// scanner spliting logic
func scanLinesFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

  delimiter := []byte{'\n','-','-','\n'}
	if i := bytes.Index(data, delimiter); i >= 0 {
		// We have a full newline-terminated line.
		return i + len(delimiter), dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

