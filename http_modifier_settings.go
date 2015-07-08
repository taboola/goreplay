package main

import (
    "errors"
    "fmt"
    "regexp"
    "strings"
    "strconv"
    "bytes"
)

//
// Handling of --http-allow-header options
//
type headerFilter struct {
    name   []byte
    regexp *regexp.Regexp
}

type HTTPHeaderFilters []headerFilter

func (h *HTTPHeaderFilters) String() string {
    return fmt.Sprint(*h)
}

func (h *HTTPHeaderFilters) Set(value string) error {
    valArr := strings.SplitN(value, ":", 2)
    if len(valArr) < 2 {
        return errors.New("need both header and value, colon-delimited (ex. user_id:^169$).")
    }
    r, err := regexp.Compile(valArr[1])
    if err != nil {
        return err
    }

    *h = append(*h, headerFilter{name: []byte(valArr[0]), regexp: r})

    return nil
}

//
// Handling of --http-allow-header-hash options
//
type headerHashFilter struct {
    name    []byte
    percent uint32
}

type HTTPHeaderHashFilters []headerHashFilter

func (h *HTTPHeaderHashFilters) String() string {
    return fmt.Sprint(*h)
}

func (h *HTTPHeaderHashFilters) Set(value string) error {
    valArr := strings.SplitN(value, ":", 2)
    if len(valArr) < 2 {
        return errors.New("need both header and value, colon-delimited (ex. user_id:1/2).")
    }

    fracArr := strings.Split(valArr[1], "/")
    if len(fracArr) < 2 {
        return errors.New("need both a numerator and denominator specified, slash-delimited (ex. user_id:1/4).")
    }

    var num, den uint64
    num, _ = strconv.ParseUint(fracArr[0], 10, 64)
    den, _ = strconv.ParseUint(fracArr[1], 10, 64)

    var f headerHashFilter
    f.name = []byte(valArr[0])
    f.percent = uint32((float64(num) / float64(den)) * 100)
    *h = append(*h, f)

    return nil
}


//
// Handling of --http-set-header option
//
type HTTPHeaders []HTTPHeader
type HTTPHeader struct {
    Name  string
    Value string
}

func (h *HTTPHeaders) String() string {
    return fmt.Sprint(*h)
}

func (h *HTTPHeaders) Set(value string) error {
    v := strings.SplitN(value, ":", 2)
    if len(v) != 2 {
        return errors.New("Expected `Key: Value`")
    }

    header := HTTPHeader{
        strings.TrimSpace(v[0]),
        strings.TrimSpace(v[1]),
    }

    *h = append(*h, header)
    return nil
}

//
// Handling of --http-allow-method option
//
type HTTPMethods [][]byte

func (h *HTTPMethods) String() string {
    return fmt.Sprint(*h)
}

func (h *HTTPMethods) Set(value string) error {
    *h = append(*h, []byte(value))
    return nil
}

func (h *HTTPMethods) Contains(value []byte) bool {
    for _, method := range *h {
        if bytes.Equal(value, method) {
            return true
        }
    }
    return false
}

//
// Handling of --http-rewrite-url option
//
type urlRewrite struct {
    src    *regexp.Regexp
    target []byte
}

type UrlRewriteMap []urlRewrite

func (r *UrlRewriteMap) String() string {
    return fmt.Sprint(*r)
}

func (r *UrlRewriteMap) Set(value string) error {
    valArr := strings.SplitN(value, ":", 2)
    if len(valArr) < 2 {
        return errors.New("need both src and target, colon-delimited (ex. /a:/b).")
    }
    regexp, err := regexp.Compile(valArr[0])
    if err != nil {
        return err
    }
    *r = append(*r, urlRewrite{src: regexp, target: []byte(valArr[1])})
    return nil
}

//
// Handling of --http-allow-url option
//
type HTTPUrlRegexp struct {
    regexp *regexp.Regexp
}

func (r *HTTPUrlRegexp) String() string {
    if r.regexp == nil {
        return ""
    }
    return r.regexp.String()
}

func (r *HTTPUrlRegexp) Set(value string) error {
    regexp, err := regexp.Compile(value)
    r.regexp = regexp
    return err
}
