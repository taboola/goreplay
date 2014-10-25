package main

import (
	"io"
	"reflect"
)

type InOutPlugins struct {
	Inputs  []io.Reader
	Outputs []io.Writer
}

type ReaderOrWriter interface{}

var Plugins *InOutPlugins = new(InOutPlugins)

// Automatically detects type of plugin and initialize it
// 
// See this article if curious about relfect stuff below: http://blog.burntsushi.net/type-parametric-functions-golang
func registerPlugin(constructor interface{}, options ...interface{}) {
	vc := reflect.ValueOf(constructor)

	vo := []reflect.Value{}
	for _, i := range options {
		vo = append(vo, reflect.ValueOf(i))
	}

	// Here we calling our constructor with list of passed options
	plugin := vc.Call(vo)[0].Interface()

	if p, ok := plugin.(io.Reader); ok {
		Plugins.Inputs = append(Plugins.Inputs, p)
	}

	if p, ok := plugin.(io.Writer); ok {
		Plugins.Outputs = append(Plugins.Outputs, p)
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
		registerPlugin(NewHTTPOutput, options, Settings.outputHTTPHeaders, Settings.outputHTTPMethods, Settings.outputHTTPUrlRegexp, Settings.outputHTTPHeaderFilters, Settings.outputHTTPHeaderHashFilters, Settings.outputHTTPElasticSearch, Settings.outputHTTPUrlRewrite)
	}
}
