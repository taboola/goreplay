package main

import (
	"regexp"
)

type HTTPUrlRegexp struct {
	regexp *regexp.Regexp
}

func (r *HTTPUrlRegexp) String() string {
	if r.regexp == nil {
		return ""
	}
	return r.regexp.String()
}

func (r *HTTPUrlRegexp) Set(value string) error {
	regexp, err := regexp.Compile(value)
	r.regexp = regexp
	return err
}
