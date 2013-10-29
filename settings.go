package gor

import (
	"flag"
	"log"
)

type AppSettings struct {
	verbose bool

	inputDummy  MultiOption
	outputDummy MultiOption

	inputTCP  MultiOption
	outputTCP MultiOption

	inputFile  MultiOption
	outputFile MultiOption

	inputRAW MultiOption

	outputHTTP MultiOption
}

var Setttings AppSettings = AppSettings{}

func init() {
	flag.BoolVar(&Setttings.verbose, "verbose", false, "")

	flag.Var(&Setttings.inputDummy, "input-dummy", "")
	flag.Var(&Setttings.outputDummy, "output-dummy", "")

	flag.Var(&Setttings.inputTCP, "input-tcp", "")
	flag.Var(&Setttings.outputTCP, "output-tcp", "")

	flag.Var(&Setttings.inputFile, "input-file", "")
	flag.Var(&Setttings.outputFile, "output-file", "")

	flag.Var(&Setttings.inputRAW, "input-raw", "")

	flag.Var(&Setttings.outputHTTP, "output-http", "")
}

func Debug(args ...interface{}) {
	if Setttings.verbose {
		log.Println(args...)
	}
}
