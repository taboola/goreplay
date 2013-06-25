package replay

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"sync"
	"time"
)

const STATS_BUF_SECONDS = 61
const STATS_ELEM_SIZE = 150

// We store stats for last 60 seconds.
// Number 60 used because http timeout limit is set to 60 seconds.
// So stats older then 60 seconds is can't be changed.
type SiteStats struct {
	Start  int64
	Update int64

	stats []RequestStat

	file *os.File

	mutex sync.Mutex
}

func NewSiteStats() (stats *SiteStats) {
	stats = &SiteStats{}
	stats.Start = time.Now().Unix()
	stats.Update = time.Now().Unix()
	stats.stats = make([]RequestStat, STATS_BUF_SECONDS)
	stats.stats[0].reset()

	stats.RotateStats()

	if Settings.StatPath != "" {
		var err error
		stats.file, err = os.OpenFile(Settings.StatPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)

		if err != nil {
			fmt.Println("ERROR: Can't write stats to ", Settings.StatPath, err)
		}
	}

	return
}

func (s *SiteStats) GetCurrent() (rs *RequestStat, idx int) {
	idx = int(s.Update-s.Start) % cap(s.stats)
	rs = &s.stats[idx]

	return
}

func (s *SiteStats) FindByTime(ts int64) *RequestStat {
	for _, stat := range s.stats {
		if stat.timestamp == ts {
			return &stat
		}
	}

	return nil
}

func (s *SiteStats) RotateStats() {
	go func() {
		for {
			time.Sleep(1 * time.Second)

			s.WriteStats()

			s.UpdateCurrent()
		}
	}()
}

func (s *SiteStats) UpdateCurrent() {
	if s.Update != time.Now().Unix() {
		s.Update = time.Now().Unix()
		rs, _ := s.GetCurrent()
		rs.reset()
	}
}

// Every second flush stats to disk
func (s *SiteStats) WriteStats() {
	if s.file == nil {
		return
	}

	// On every write, we rewriting whole stats
	_, err := s.file.Seek(-STATS_ELEM_SIZE*(STATS_BUF_SECONDS-1), 2)

	// If file is too small (less then 60 records), just seek to start of the file
	if err != nil {
		s.file.Seek(0, 0)
	}

	_, currIdx := s.GetCurrent()

	for i := cap(s.stats); i >= 0; i-- {
		idx := currIdx - i

		if idx < 0 {
			idx = cap(s.stats) + idx
		}

		rs := s.stats[idx]

		if (s.Update - s.Start) > int64(idx) {
			if rs.Count > 0 {
				Debug("Writing:", idx, rs)
			}

			s.file.Write(rs.Encode())
		}
	}
}

func (s *SiteStats) IncReq() {
	s.UpdateCurrent()

	rs, _ := s.GetCurrent()
	rs.IncReq()
}

func (s *SiteStats) IncResp(resp *HttpResponse) {
	rs := s.FindByTime(resp.req.created / int64(time.Second))

	rs.IncResp(resp)
}

func (s *SiteStats) Count() int {
	rs, _ := s.GetCurrent()
	return rs.Count
}

// Stats stores in context of current timestamp
type RequestStat struct {
	timestamp int64

	Codes map[int]int // { 200: 10, 404:2, 500:1 }

	Count  int // All requests including errors
	Errors int // Rquests with errors (timeout or host not reachable). Not include 50x errors.

	mutex sync.Mutex
}

// Called on request start
func (s *RequestStat) IncReq() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Count++
}

// Called after response
func (s *RequestStat) IncResp(resp *HttpResponse) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if resp.err != nil {
		s.Codes[509]++
	} else {
		s.Codes[resp.resp.StatusCode]++
	}
}

// Updated stats timestamp to current time and reset to zero all stats values
// TODO: Further on reset it should write stats to file
func (s *RequestStat) reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.timestamp = time.Now().Unix()

	s.Codes = make(map[int]int)
	s.Count = 0
}

func (s *RequestStat) Encode() []byte {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	enc.Encode(s)
	encoded := data.Bytes()

	// Each record should be exactly 150 bytes long
	for i := (STATS_ELEM_SIZE - len(encoded)); i != 0; i-- {
		data.WriteByte(0)
	}

	return data.Bytes()
}

// Called on request start
func (s *RequestStat) String() string {
	return fmt.Sprint("Requests:", s.Count, "Status codes:", s.Codes)
}
