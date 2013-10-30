package gor

import (
	"flag"
	"log"
)

type AppSettings struct {
	verbose bool

	splitOutput bool

	inputDummy  MultiOption
	outputDummy MultiOption

	inputTCP  MultiOption
	outputTCP MultiOption

	inputFile  MultiOption
	outputFile MultiOption

	inputRAW MultiOption

	outputHTTP        MultiOption
	outputHTTPHeaders HTTPHeaders
}

var Settings AppSettings = AppSettings{}

func init() {
	flag.BoolVar(&Settings.verbose, "verbose", false, "")

	flag.BoolVar(&Settings.splitOutput, "split-output", false, "")

	flag.Var(&Settings.inputDummy, "input-dummy", "")
	flag.Var(&Settings.outputDummy, "output-dummy", "")

	flag.Var(&Settings.inputTCP, "input-tcp", "")
	flag.Var(&Settings.outputTCP, "output-tcp", "")

	flag.Var(&Settings.inputFile, "input-file", "")
	flag.Var(&Settings.outputFile, "output-file", "")

	flag.Var(&Settings.inputRAW, "input-raw", "")

	flag.Var(&Settings.outputHTTP, "output-http", "")

	flag.Var(&Settings.outputHTTPHeaders, "output-http-header", "")
}

func Debug(args ...interface{}) {
	if Settings.verbose {
		log.Println(args...)
	}
}
