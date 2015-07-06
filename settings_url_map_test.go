package main

import (
    "testing"
)

func TestUrlRewriteMap(t *testing.T) {
    var err error
    rewrites := UrlRewriteMap{}

    if err = rewrites.Set("/v1/user/([^\\/]+)/ping:/v2/user/$1/ping"); err != nil {
        t.Error("Should set mapping", err)
    }

    if err = rewrites.Set("/v1/user/([^\\/]+)/ping"); err == nil {
        t.Error("Should not set mapping without :")
    }
}