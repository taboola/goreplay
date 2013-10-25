package gor

import (
	"flag"
)

type AppSettings struct {
	inputDummy  MultiOption
	outputDummy MultiOption

	inputRAW MultiOption

	inputTCP  MultiOption
	outputTCP MultiOption

	outputHTTP MultiOption
}

var Setttings AppSettings = AppSettings{}

func init() {
	flag.Var(&Setttings.inputDummy, "input-dummy", "")
	flag.Var(&Setttings.outputDummy, "output-dummy", "")

	flag.Var(&Setttings.inputRAW, "input-raw", "")

	flag.Var(&Setttings.inputTCP, "input-tcp", "")
	flag.Var(&Setttings.outputTCP, "output-tcp", "")

	flag.Var(&Setttings.outputHTTP, "output-http", "")
}
