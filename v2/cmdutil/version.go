package cmdutil

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// The Build* variables are used by NewVersionCommand and NewRootCommand. They
// should be overwritten on build time by using ldflags.
var (
	BuildVersion     = "unknown"
	BuildPackage     = "unknown"
	BuildDate        = "unknown"
	BuildHash        = "unknown"
	BuildEnvironment = "unknown"
	BuildName        = "unknown"
)

// NewVersionCommand creates a Cobra command, which prints the version
// and other build parameters (see Build* variables) and exits.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "version",
		Short:             "shows version of this application",
		PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("name:        %s\n", BuildName)
			fmt.Printf("package:     %s\n", BuildPackage)
			fmt.Printf("version:     %s\n", BuildVersion)
			fmt.Printf("build date:  %s\n", BuildDate)
			fmt.Printf("scm hash:    %s\n", BuildHash)
			fmt.Printf("environment: %s\n", BuildEnvironment)
		},
	}

	return cmd
}

func WithVersionCommand() Option {
	return func(cmd *cobra.Command) error {
		cmd.AddCommand(NewVersionCommand())
		return nil
	}
}

func WithVersionLog(level logrus.Level) Option {
	return func(cmd *cobra.Command) error {
		cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
			logrus.WithFields(logrus.Fields{
				"Version": BuildVersion,
				"Date":    BuildDate,
				"Commit":  BuildHash,
			}).Logf(level, "%s started", BuildName)
		}
		return nil
	}
}
