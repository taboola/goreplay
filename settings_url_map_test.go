package main

import (
	"testing"
)

func TestUrlRewriteMap_1(t *testing.T) {
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

func TestUrlRewriteMap_2(t *testing.T) {
	var url string

	rewrites := UrlRewriteMap{}

	err := rewrites.Set("/abc?\\d{4}5$:/123")
	if err != nil {
		t.Error("Should not error on /abc?\\d{4}:/123")
	}

	url = "/ab12345"
	if rewrites.Rewrite(url) == url {
		t.Error("Request url should have been rewritten, wasn't")
	}

	url = "/ab"
	if rewrites.Rewrite(url) != url {
		t.Error("Request url should not have been rewritten, was")
	}
}
