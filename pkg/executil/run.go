package executil

import (
	"context"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Run starts the specified command and waits for it to complete.
//
// The difference to Run from exec.CommandContext is that it sends an interrupt
// instead of a kill, which gives the process time for a graceful shutdown.
func Run(ctx context.Context, cmd *exec.Cmd) error {
	commandline := strings.Join(cmd.Args, " ")
	logrus.WithFields(logrus.Fields{
		"Args": cmd.Args,
		"Dir":  cmd.Dir,
	}).Debugf("running command `%s`", commandline)

	err := cmd.Start()
	if err != nil {
		return errors.WithStack(err)
	}

	done := make(chan struct{}, 1)
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			logrus.Debugf("sending interrupt signal to `%s`", commandline)
			cmd.Process.Signal(syscall.SIGINT)
		case <-done:
			// This mean wait() already exited and we can stop to wait for the
			// cancelation.
		}
	}()

	return errors.Wrapf(cmd.Wait(), "failed to run `%s`", commandline)
}
