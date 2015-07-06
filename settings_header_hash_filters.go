package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
