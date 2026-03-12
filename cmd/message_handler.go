package cmd

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fatih/color"
)

func failOnFprintError(_ int, err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type messageHandler struct {
	exit         func(int)
	stdoutWriter io.Writer
	stderrWriter io.Writer
}

func newMessageHandler() messageHandler {
	return messageHandler{os.Exit, os.Stdout, os.Stderr}
}

func (m messageHandler) print(str string, args ...any) {
	failOnFprintError(fmt.Fprintf(m.stdoutWriter, str, args...))
}

func (m messageHandler) errorFatal(err error) {
	failOnFprintError(color.New(color.FgRed).Fprint(m.stderrWriter, err.Error()+"\n"))
	m.exit(1)
}

func (m messageHandler) error(err error) {
	failOnFprintError(color.New(color.FgRed).Fprint(m.stderrWriter, err.Error()+"\n"))
}

func (m messageHandler) errorFatalStr(err string) {
	failOnFprintError(color.New(color.FgRed).Fprint(m.stderrWriter, err+"\n"))
	m.exit(1)
}

func (m messageHandler) success(str string, args ...any) {
	failOnFprintError(color.New(color.FgGreen).Fprintf(m.stdoutWriter, str+"\n", args...))
}
