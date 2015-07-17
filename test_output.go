package main

type writeCallback func(data []byte)

// TestOutput used in testing to intercept any output into callback
type TestOutput struct {
	cb writeCallback
}

// NewTestOutput constructor for TestOutput, accepts callback which get called on each incoming Write
func NewTestOutput(cb writeCallback) (i *TestOutput) {
	i = new(TestOutput)
	i.cb = cb

	return
}

func (i *TestOutput) Write(data []byte) (int, error) {
	i.cb(data)

	return len(data), nil
}

func (i *TestOutput) String() string {
	return "Test Input"
}
