package main

import (
	"testing"
	"net/http"
)

func TestUrlRewriteMap(t *testing.T) {
	rewrites := UrlRewriteMap{}

	err := rewrites.Set("/abc:/123")
	if err != nil {
		t.Error("Should not error on /abc:/123")
	}

	req := http.Request{}
        req.URL.Path = "/abc"

        if(rewrites.Rewrite(req.URL.Path) == req.URL.Path) {
                t.Error("Request url should have been rewritten, wasn't")
        }

        req.URL.Path = "/wibble"
        if(rewrites.Rewrite(req.URL.Path) != req.URL.Path) {
                t.Error("Request url should not have been rewritten, was")
        }
}
