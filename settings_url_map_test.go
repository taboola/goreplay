package main

import (
	"testing"
)

func TestUrlRewriteMap(t *testing.T) {
	var url string

	rewrites := UrlRewriteMap{}

	err := rewrites.Set("/abc:/123")
	if err != nil {
		t.Error("Should not error on /abc:/123")
	}

	url = "/abc"

	if rewrites.Rewrite(url) == url {
		t.Error("Request url should have been rewritten, wasn't")
	}

	url = "/wibble"
	if rewrites.Rewrite(url) != url {
		t.Error("Request url should not have been rewritten, was")
	}
}
