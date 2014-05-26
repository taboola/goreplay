package main

import (
	"regexp"
	"net/http"
	"fmt"
	"strings"
	"errors"
	"log"
)

type headerFilter struct {
	name   string
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
		return errors.New("could not compile regexp.")
	}

	*h = append(*h, headerFilter{name: valArr[0], regexp: r})

	return nil
}

func (h *HTTPHeaderFilters) Good(req *http.Request) bool {
	for _, f := range *h {
		log.Print("Header: ", f.name, req.Header.Get(f.name))
		if !f.regexp.Match([]byte(req.Header.Get(f.name))) {
			log.Println("Failing request on header regex")
			return false
		}
	} 
	return true
}