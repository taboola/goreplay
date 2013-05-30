package replay

import (
    "log"
    "time"
)

type RequestStat struct {
    timestamp int64

    codes map[int]int

    count  int
    errors int
}

func (s *RequestStat) inc(resp *Response) {
    if s.timestamp != time.Now().Unix() {
        s.reset()
    }

    if resp.err != nil {
        s.errors++
        return
    }

    s.count++
    s.codes[resp.resp.StatusCode]++
}

func (s *RequestStat) reset() {
    log.Println("reseting stats", s)

    s.timestamp = time.Now().Unix()

    s.codes = make(map[int]int)
    s.count = 0
    s.errors = 0
}
