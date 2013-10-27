package gor

import (
	"io"
)

type InOutPlugins struct {
	Inputs  []io.Reader
	Outputs []io.Writer
}

var Plugins *InOutPlugins = new(InOutPlugins)

func InitPlugins() {
	for _, options := range Setttings.inputDummy {
		Plugins.Inputs = append(Plugins.Inputs, NewDummyInput(options))
	}

	for _, options := range Setttings.outputDummy {
		Plugins.Outputs = append(Plugins.Outputs, NewDummyOutput(options))
	}

	for _, options := range Setttings.inputRAW {
		Plugins.Inputs = append(Plugins.Inputs, NewRAWInput(options))
	}

	for _, options := range Setttings.inputTCP {
		Plugins.Inputs = append(Plugins.Inputs, NewTCPInput(options))
	}

	for _, options := range Setttings.outputTCP {
		Plugins.Outputs = append(Plugins.Outputs, NewTCPOutput(options))
	}

	for _, options := range Setttings.outputFile {
		Plugins.Outputs = append(Plugins.Outputs, NewFileOutput(options))
	}

	for _, options := range Setttings.outputHTTP {
		Plugins.Outputs = append(Plugins.Outputs, NewHTTPOutput(options))
	}
}
