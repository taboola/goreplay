package main

import (
	"testing"
)

func TestHTTPMethods(t *testing.T) {
	methods := HTTPMethods{}

	methods.Set("lower")
	methods.Set("UPPER")

	if !methods.Contains([]byte("LOWER")) {
		t.Error("Does not contain LOWER")
	}

	if !methods.Contains([]byte("UPPER")) {
		t.Error("Does not contain UPPER")
	}

	if methods.Contains([]byte("ABSENT")) {
		t.Error("Does contain ABSENT")
	}
}
