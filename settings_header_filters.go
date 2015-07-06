package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

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
