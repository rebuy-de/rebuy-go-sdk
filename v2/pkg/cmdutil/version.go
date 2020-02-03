package cmdutil

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// The Build* variables are used by NewVersionCommand and NewRootCommand. They
// should be overwritten on build time by using ldflags.
var (
	Name       = "unknown"
	Version    = "unknown"
	GoModule   = "unknown"
	GoPackage  = "unknown"
	BuildDate  = "unknown"
	CommitDate = "unknown"
	CommitHash = "unknown"
)

// NewVersionCommand creates a Cobra command, which prints the version
// and other build parameters (see Build* variables) and exits.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "version",
		Short:             "Shows version of this application",
		PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Name:       %s\n", Name)
			fmt.Printf("Version:    %s\n", Version)
			fmt.Printf("GoModule:   %s\n", GoModule)
			fmt.Printf("GoPackage:  %s\n", GoPackage)
			fmt.Printf("BuildDate:  %s\n", BuildDate)
			fmt.Printf("CommitDate: %s\n", CommitDate)
			fmt.Printf("CommitHash: %s\n", CommitHash)
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
				"Version": Version,
				"Date":    CommitDate,
				"Commit":  CommitHash,
			}).Logf(level, "%s started", Name)
		}
		return nil
	}
}
