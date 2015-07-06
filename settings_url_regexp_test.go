package main

import (
	"testing"
)

func TestHTTPUrlRegexp(t *testing.T) {
	filter := HTTPUrlRegexp{}
	filter.Set("^www.google.com/admin/")
}
