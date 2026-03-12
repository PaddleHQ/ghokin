package cmd_test

import (
	"bytes"
	"testing"

	"github.com/PaddleHQ/ghokin/v4/cmd"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestFormat(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	msgHandler := cmd.NewTestMessageHandler(
		func(exitCode int) {
			panic(exitCode)
		},
		&stdout,
		&stderr,
	)

	c := &cobra.Command{}
	args := []string{}

	cmd.TestFormat(msgHandler, c, args)

	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}
