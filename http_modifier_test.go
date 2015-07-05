package main

import (
    "testing"
)

func TestHTTPModifierWithoutConfig(t *testing.T) {
    if NewHTTPModifier(&HTTPModifierConfig{}) != nil {
        t.Error("If no config specified should not be initialized")
    }
}

func TestHTTPModifierHeaderFilters(t *testing.T) {
    filters := HTTPHeaderFilters{}
    filters.Set("Host:^www.w3.org$")

    modifier := NewHTTPModifier(&HTTPModifierConfig{
        headerFilters: filters,
    })

    payload := []byte("POST /post HTTP/1.1\r\nContent-Length: 7\r\nHost: www.w3.org\r\n\r\na=1&b=2")

    if len(modifier.Rewrite(payload)) == 0 {
        t.Error("Request should pass filters")
    }

    filters = HTTPHeaderFilters{}
    // Setting filter that not match our header
    filters.Set("Host:^www.w4.org$")

    modifier = NewHTTPModifier(&HTTPModifierConfig{
        headerFilters: filters,
    })

    if len(modifier.Rewrite(payload)) != 0 {
        t.Error("Request should not pass filters")
    }
}
