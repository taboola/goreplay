package main

import (
	"time"
	"log"
	"strconv"
)

const (
	rate = int64(5)
)

type GorStat struct {
	statName	string
	latest		int
	min			int
	max			int

	currentTime int64
}

func NewGorStat(statName string) GorStat {
	s := new(GorStat)
	s.statName = statName
	s.latest = 0
	s.min = 0
	s.max = 0

	if Settings.stats {
		go s.reportStats()
	}
	return s
}

func (s *GorStat) Write(latest int) {
	if Settings.stats {
		if latest > s.max {
			s.max = latest
		}
		if latest < s.min {
			s.min = latest
		}
		s.latest = latest
	}
}

func (s *GorStat) String() string {
	return s.statName + ":" + strconv.Itoa(s.latest) + "," + strconv.Itoa(s.min) + "," + strconv.Itoa(s.max)
}

func (s *GorStat) reportStats() {
	if (time.Now().UnixNano() - s.currentTime) > (rate * time.Second.Nanoseconds()) {
		s.currentTime = time.Now().UnixNano()
		log.Println(s)
	}
}

