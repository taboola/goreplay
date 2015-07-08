package main

import (
    "testing"
)

func TestHTTPHeaderFilters(t *testing.T) {
    filters := HTTPHeaderFilters{}

    err := filters.Set("Header1:^$")
    if err != nil {
        t.Error("Should not error on Header1:^$")
    }

    err = filters.Set("Header2:^:$")
    if err != nil {
        t.Error("Should not error on Header2:^:$")
    }

    // Missing colon
    err = filters.Set("Header3-^$")
    if err == nil {
        t.Error("Should error on Header2:^:$")
    }
}

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

func TestHTTPMethods(t *testing.T) {
    methods := HTTPMethods{}

    methods.Set("GET")
    methods.Set("POST")

    if !methods.Contains([]byte("GET")) {
        t.Error("Does not contain GET")
    }

    if !methods.Contains([]byte("POST")) {
        t.Error("Does not contain POST")
    }
}

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

