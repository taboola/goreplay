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

	err = filters.Set("Header2:1/2")
	if err != nil {
		t.Error("Should not error on Header2:^:$")
	}

    // Denominator must be power of 2
	err = filters.Set("HeaderIrrelevant:1/3")
	if err == nil {
		t.Error("Should error on HeaderIrrelevant:1/3")
	}

    // Denominator must be power of 2
	err = filters.Set("Pow2Denom:1/31")
	if err == nil {
		t.Error("Should error on Pow2Denom:1/31")
	}
}
