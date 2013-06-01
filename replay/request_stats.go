package replay

import (
    "log"
    "time"
)

type RequestStat struct {
    timestamp int64

    Codes map[int]int

    Count  int
    Errors int

    host *ForwardHost
}

func (s *RequestStat) Touch() {
    if s.timestamp != time.Now().Unix() {
        s.reset()
    }
}

func (s *RequestStat) IncReq() {
    s.Touch()

    s.Count++
}

func (s *RequestStat) IncResp(resp *HttpResponse) {
    s.Touch()

    if resp.err != nil {
        s.Errors++
        return
    }

    s.Codes[resp.resp.StatusCode]++
}

func (s *RequestStat) reset() {
    if s.timestamp != 0 {
        log.Println("Host:", s.host.Url, "Requests:", s.Count, "Errors:", s.Errors, "Status codes:", s.Codes)
    }

    s.timestamp = time.Now().Unix()

    s.Codes = make(map[int]int)
    s.Count = 0
    s.Errors = 0
}

func NewRequestStats(host *ForwardHost) (stat *RequestStat) {
    stat = &RequestStat{host: host}
    stat.reset()

    return
}
