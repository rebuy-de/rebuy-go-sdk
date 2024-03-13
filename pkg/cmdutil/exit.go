package cmdutil

import (
	"os"

	log "github.com/sirupsen/logrus"
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
//
// Deprecated: Bubble the error up to the Runner.Run function and return it
// there instead. It is still preferable to let the application die, when there
// is no obvious way of handling it, but in reality this is not often the case
// and Must is encouraging permature exits.
func Must(err error) {
	if err == nil {
		return
	}

	log.Debugf("%+v", err)
	log.Error(err)
	Exit(ExitCodeGeneralError)
}
