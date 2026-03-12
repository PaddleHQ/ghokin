package cmd

import "io"

// NewTestMessageHandler creates a messageHandler for testing from the cmd_test package.
func NewTestMessageHandler(exitFn func(int), stdout, stderr io.Writer) messageHandler {
	return messageHandler{exit: exitFn, stdoutWriter: stdout, stderrWriter: stderr}
}

// Test exports for cmd_test package.
var (
	TestCheck            = check
	TestFormat           = format
	TestFormatAndReplace = formatAndReplace
	TestFormatOnStdout   = formatOnStdout
	TestInitConfig       = initConfig
)

// SetTestCfgFile sets cfgFile for testing.
func SetTestCfgFile(s string) { cfgFile = s }
