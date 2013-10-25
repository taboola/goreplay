package gor

import (
	"flag"
)

type AppSettings struct {
	inputDummy  MultiOption
	outputDummy MultiOption

	inputTCP  MultiOption
	outputTCP MultiOption
}

var Setttings AppSettings = AppSettings{}

func init() {
	flag.Var(&Setttings.inputDummy, "input-dummy", "")
	flag.Var(&Setttings.outputDummy, "output-dummy", "")

	flag.Var(&Setttings.inputTCP, "input-tcp", "")
	flag.Var(&Setttings.outputTCP, "output-tcp", "")
}
