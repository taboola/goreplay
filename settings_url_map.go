package main

import (
	"errors"
	"fmt"
	"strings"
)

type urlRewrite struct {
	src    string
	target string
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
	*r = append(*r, urlRewrite{src: valArr[0], target: valArr[1]})
	return nil
}

func (r *UrlRewriteMap) Rewrite(path string) string {
	for _, f := range *r {
		if f.src == path {
			return f.target
		}
	}
	return path
}
