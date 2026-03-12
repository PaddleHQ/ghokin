package cmd_test

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/PaddleHQ/ghokin/v4/cmd"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestFormatOnStdoutFromFile(t *testing.T) {
	var code int
	var w sync.WaitGroup
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	viper.Set("indent", 2)

	msgHandler := cmd.NewTestMessageHandler(
		func(exitCode int) {
			panic(exitCode)
		},
		&stdout,
		&stderr,
	)

	w.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				code = r.(int)
			}

			w.Done()
		}()

		c := &cobra.Command{}
		args := []string{"fixtures/feature.feature"}

		cmd.TestFormatOnStdout(msgHandler, c, args)
	}()

	w.Wait()

	b, err := os.ReadFile("fixtures/feature.feature")

	assert.NoError(t, err)

	assert.Equal(t, 0, code, "Must exit with no errors (exit 0)")
	assert.Equal(t, string(b), stdout.String())
}

func TestFormatOnStdoutFromStdin(t *testing.T) {
	var code int
	var w sync.WaitGroup
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	viper.Set("indent", 2)

	msgHandler := cmd.NewTestMessageHandler(
		func(exitCode int) {
			panic(exitCode)
		},
		&stdout,
		&stderr,
	)

	w.Add(1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				code = r.(int)
			}

			w.Done()
		}()

		content, err := os.ReadFile("fixtures/feature.feature")
		assert.NoError(t, err)
		c := &cobra.Command{}
		args := []string{}
		c.SetIn(bytes.NewBuffer(content))
		cmd.TestFormatOnStdout(msgHandler, c, args)
	}()

	w.Wait()

	b, err := os.ReadFile("fixtures/feature.feature")

	assert.NoError(t, err)

	assert.Equal(t, 0, code, "Must exit with no errors (exit 0)")
	assert.Equal(t, string(b), stdout.String())
}

func TestFormatOnStdoutWithErrors(t *testing.T) {
	var code int
	var w sync.WaitGroup
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	msgHandler := cmd.NewTestMessageHandler(
		func(exitCode int) {
			panic(exitCode)
		},
		&stdout,
		&stderr,
	)

	type scenario struct {
		args   []string
		errMsg string
	}

	scenarios := []scenario{
		{
			[]string{"fixtures/featurefeature.feature"},
			"failed to read file: open fixtures/featurefeature.feature: no such file or directory\n",
		},
	}

	for _, s := range scenarios {
		w.Add(1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					code = r.(int)
				}

				w.Done()
			}()

			c := &cobra.Command{}

			cmd.TestFormatOnStdout(msgHandler, c, s.args)
		}()

		w.Wait()

		assert.Equal(t, 1, code, "Must exit with errors (exit 1)")
		assert.Equal(t, s.errMsg, stderr.String())

		stderr.Reset()
		stdout.Reset()
	}
}
