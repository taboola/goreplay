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