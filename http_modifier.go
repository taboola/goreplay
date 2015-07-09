package main

import (
	"github.com/buger/gor/proto"
	"hash/fnv"
)

type HTTPModifier struct {
	config *HTTPModifierConfig
}

func NewHTTPModifier(config *HTTPModifierConfig) *HTTPModifier {
	// Optimization to skip modifier completely if we do not need it
	if len(config.urlRegexp) == 0 &&
		len(config.urlNegativeRegexp) == 0 &&
		len(config.urlRewrite) == 0 &&
		len(config.headerFilters) == 0 &&
		len(config.headerHashFilters) == 0 &&
		len(config.paramHashFilters) == 0 &&
		len(config.headers) == 0 &&
		len(config.methods) == 0 {
		return nil
	}

	return &HTTPModifier{config: config}
}

func (m *HTTPModifier) Rewrite(payload []byte) (response []byte) {
	if len(m.config.methods) > 0 && !m.config.methods.Contains(proto.Method(payload)) {
		return
	}

	if len(m.config.urlRegexp) > 0 {
		path := proto.Path(payload)

		matched := false

		for _, f := range m.config.urlRegexp {
			if f.regexp.Match(path) {
				matched = true
			}
		}

		if !matched {
			return
		}
	}

	if len(m.config.urlNegativeRegexp) > 0 {
		path := proto.Path(payload)

		for _, f := range m.config.urlNegativeRegexp {
			if f.regexp.Match(path) {
				return
			}
		}
	}

	if len(m.config.headerFilters) > 0 {
		for _, f := range m.config.headerFilters {
			value, s, _, _ := proto.Header(payload, f.name)

			if s != -1 && !f.regexp.Match(value) {
				return
			}
		}
	}

	if len(m.config.headerHashFilters) > 0 {
		for _, f := range m.config.headerHashFilters {
			value, s, _, _ := proto.Header(payload, f.name)

			if s != -1 {
				hasher := fnv.New32a()
				hasher.Write(value)

				if (hasher.Sum32() % 100) >= f.percent {
					return
				}
			}
		}
	}

	if len(m.config.paramHashFilters) > 0 {
		for _, f := range m.config.paramHashFilters {
			value, s, _ := proto.PathParam(payload, f.name)

			if s != -1 {
				hasher := fnv.New32a()
				hasher.Write(value)

				if (hasher.Sum32() % 100) >= f.percent {
					return
				}
			}
		}
	}

	if len(m.config.urlRewrite) > 0 {
		path := proto.Path(payload)

		for _, f := range m.config.urlRewrite {
			if f.src.Match(path) {
				path = f.src.ReplaceAll(path, f.target)
				payload = proto.SetPath(payload, path)

				break
			}
		}
	}

	if len(m.config.headers) > 0 {
		for _, header := range m.config.headers {
			payload = proto.SetHeader(payload, []byte(header.Name), []byte(header.Value))
		}
	}

	return payload
}
