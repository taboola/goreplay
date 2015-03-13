package main

import (
	"io"
	"reflect"
	"strings"
)

type InOutPlugins struct {
	Inputs  []io.Reader
	Outputs []io.Writer
}

type ReaderOrWriter interface{}

var Plugins *InOutPlugins = new(InOutPlugins)

func extractLimitOptions(options string) (string, string) {
	split := strings.Split(options, "|")

	if len(split) > 1 {
		return split[0], split[1]
	} else {
		return split[0], ""
	}
}

// Automatically detects type of plugin and initialize it
//
// See this article if curious about relfect stuff below: http://blog.burntsushi.net/type-parametric-functions-golang
func registerPlugin(constructor interface{}, options ...interface{}) {
	vc := reflect.ValueOf(constructor)

	// Pre-processing options to make it work with reflect
	vo := []reflect.Value{}
	for _, oi := range options {
		vo = append(vo, reflect.ValueOf(oi))
	}

	// Removing limit options from path
	path, limit := extractLimitOptions(vo[0].String())

	// Writing value back without limiter "|" options
	vo[0] = reflect.ValueOf(path)

	// Calling our constructor with list of given options
	plugin := vc.Call(vo)[0].Interface()
	plugin_wrapper := plugin

	if limit != "" {
		plugin_wrapper = NewLimiter(plugin, limit)
	} else {
		plugin_wrapper = plugin
	}

	if _, ok := plugin.(io.Reader); ok {
		Plugins.Inputs = append(Plugins.Inputs, plugin_wrapper.(io.Reader))
	}

	if _, ok := plugin.(io.Writer); ok {
		Plugins.Outputs = append(Plugins.Outputs, plugin_wrapper.(io.Writer))
	}
}

func InitPlugins() {
	for _, options := range Settings.inputDummy {
		registerPlugin(NewDummyInput, options)
	}

	for _, options := range Settings.outputDummy {
		registerPlugin(NewDummyOutput, options)
	}

	for _, options := range Settings.inputRAW {
		registerPlugin(NewRAWInput, options)
	}

	for _, options := range Settings.inputTCP {
		registerPlugin(NewTCPInput, options)
	}

	for _, options := range Settings.outputTCP {
		registerPlugin(NewTCPOutput, options)
	}

	for _, options := range Settings.inputFile {
		registerPlugin(NewFileInput, options)
	}

	for _, options := range Settings.outputFile {
		registerPlugin(NewFileOutput, options)
	}

	for _, options := range Settings.inputHTTP {
		registerPlugin(NewHTTPInput, options)
	}

	for _, options := range Settings.outputHTTP {
		registerPlugin(NewHTTPOutput, options, Settings.outputHTTPHeaders, Settings.outputHTTPMethods, Settings.outputHTTPUrlRegexp, Settings.outputHTTPHeaderFilters, Settings.outputHTTPHeaderHashFilters, Settings.outputHTTPElasticSearch, Settings.outputHTTPUrlRewrite, Settings.outputHTTPRedirects)
	}
}
