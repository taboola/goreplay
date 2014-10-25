package main

import (
	"io"
	"testing"
)

func TestPluginsRegistration(t *testing.T) {
	Plugins.Inputs = []io.Reader{}
	Plugins.Outputs = []io.Writer{}

	Settings.inputDummy = MultiOption{"[]"}
	Settings.outputDummy = MultiOption{"[]"}
	Settings.inputFile = MultiOption{"/dev/null"}

	InitPlugins()

	if len(Plugins.Inputs) != 2 {
		t.Errorf("Should be 2 inputs")
	}

	if _, ok := Plugins.Inputs[0].(*DummyInput); !ok {
		t.Errorf("First input should be DummyInput")
	}

	if _, ok := Plugins.Inputs[1].(*FileInput); !ok {
		t.Errorf("Second input should be FileInput")
	}

	if len(Plugins.Outputs) != 1 {
		t.Errorf("Should be 1 output")
	}

	if _, ok := Plugins.Outputs[0].(*DummyOutput); !ok {
		t.Errorf("Output should be DummyOutput")
	}
}
