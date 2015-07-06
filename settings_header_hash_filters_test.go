package main

import (
	"testing"
)

func TestHTTPHeaderHashFilters(t *testing.T) {
	filters := HTTPHeaderHashFilters{}

	err := filters.Set("Header1:1/2")
	if err != nil {
		t.Error("Should not error on Header1:^$")
	}

	err = filters.Set("Header2:1")
	if err == nil {
		t.Error("Should error on Header2:^:$")
	}
}
