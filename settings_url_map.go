package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

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
	*r = append(*r, urlRewrite{src: regexp, target: []byte(valArr[1]) })
	return nil
}