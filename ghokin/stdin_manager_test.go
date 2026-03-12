package ghokin_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/PaddleHQ/ghokin/v4/ghokin"

	"github.com/stretchr/testify/assert"
)

type failingReader struct{}

func (f failingReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("an error occurred when reading data")
}

func TestStdinManagerTransform(t *testing.T) {
	type scenario struct {
		name  string
		setup func() (ghokin.StdinManager, io.Reader)
		test  func([]byte, error)
	}

	scenarios := []scenario{
		{
			"Format the stream from stdin on stdout",
			func() (ghokin.StdinManager, io.Reader) {
				stdinManager := ghokin.NewStdinManager(
					2,
					map[string]string{
						"seq": "seq 1 3",
					},
				)
				content, err := os.ReadFile("fixtures/file1.feature")
				assert.NoError(t, err)
				return stdinManager, bytes.NewBuffer(content)
			},
			func(buf []byte, _ error) {
				b, e := os.ReadFile("fixtures/file1.feature")
				assert.NoError(t, e)
				assert.Equal(t, string(b), string(buf))
			},
		},
		{
			"Format a stream from stdin fails because reading the stdin stream fails",
			func() (ghokin.StdinManager, io.Reader) {
				stdinManager := ghokin.NewStdinManager(
					2,
					map[string]string{},
				)
				return stdinManager, failingReader{}
			},
			func(_ []byte, err error) {
				assert.Error(t, err)
			},
		},
		{
			"Format an invalid stream from stdin failed",
			func() (ghokin.StdinManager, io.Reader) {
				stdinManager := ghokin.NewStdinManager(
					2,
					map[string]string{
						"seq": "seq 1 3",
					},
				)
				content, err := os.ReadFile("fixtures/invalid.feature")
				assert.NoError(t, err)
				return stdinManager, bytes.NewBuffer(content)
			},
			func(_ []byte, err error) {
				assert.Error(t, err)
			},
		},
		{
			"Format a stream from stdin fails because of an invalid command",
			func() (ghokin.StdinManager, io.Reader) {
				stdinManager := ghokin.NewStdinManager(
					2,
					map[string]string{
						"abcdefg": "abcdefg",
					},
				)
				content, err := os.ReadFile("fixtures/invalid-cmd.feature")
				assert.NoError(t, err)
				return stdinManager, bytes.NewBuffer(content)
			},
			func(_ []byte, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			stdinManager, reader := scenario.setup()
			scenario.test(stdinManager.Transform(reader))
		})
	}
}
