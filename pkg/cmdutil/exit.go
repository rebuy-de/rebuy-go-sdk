package cmdutil

import (
	"os"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	ExitCodeOK           = 0
	ExitCodeGeneralError = 1
	ExitCodeUsage        = 2
	ExitCodeSDK          = 16
	ExitCodeCustom       = 32

	ExitCodeMultipleInterrupts = ExitCodeSDK + 0
)

type exitCode struct {
	code int
}

// Exit causes the current program to exit with the given status code. On the
// contrary to os.Exit, it respects defer statements. It requires the
// HandleExit function to be deferred in top of the main function.
//
// Internally this is done by throwing a panic with the ExitCode type, which
// gets recovered in the HandleExit function.
func Exit(code int) {
	panic(exitCode{code: code})
}

// HandleExit recovers from Exit calls and terminates the current program with
// a proper exit code. It should get deferred at the beginning of the main
// function.
func HandleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(exitCode); ok {
			os.Exit(exit.code)
		}
		panic(e) // not an Exit, bubble up
	}
}

// Must exits the application via Exit(1) and logs the error, if err does not
// equal nil. Additionally it logs the error with `%+v` to the debug log, so it
// can used together with github.com/pkg/errors to retrive more details about
// the error.
func must(err error) {
	if err == nil {
		return
	}

	log.Debugf("%+v", err)
	log.Error(err)
	Exit(ExitCodeGeneralError)
}
