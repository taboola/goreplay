package main

import (
	"regexp"
	"net/http"
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

func (r *HTTPUrlRegexp) Good(req *http.Request) bool {
	if r.regexp == nil {
		return true
	}
	return r.regexp.Match([]byte(req.Host + req.URL.String()))
}