package cmd

import (
	"context"
	"embed"
	"io/fs"
	"os"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/redisutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//go:embed assets/*
var assetFS embed.FS

// NewRootCommand initializes the cobra.Command with support of the cmdutil
// package.
func NewRootCommand() *cobra.Command {
	runner := new(Runner)

	return cmdutil.New(
		"full", "rebuy-go-sdk-full-example",
		runner.Bind,
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithLogToGraylog(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		cmdutil.WithSubCommand(cmdutil.New(
			"daemon", "Run the application",
			cmdutil.WithRun(runner.Daemon),
		)),

		cmdutil.WithSubCommand(cmdutil.New(
			"dev", "Run the application in dev mode for local development",
			cmdutil.WithRun(runner.Dev),
		)),
	)
}

// Runner bootstraps the application. It defines the related flags and calls
// the actual server code.
type Runner struct {
	redisAddress string
}

// Bind defines command line flags and stores the results in the Runner struct.
func (r *Runner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.redisAddress, "redis-address", "",
		`Address of the Redis instance.`)
	return nil
}

// Daemon initializes the server with production-ready settings.
func (r *Runner) Daemon(ctx context.Context, cmd *cobra.Command, args []string) {
	var (
		redisPrefix = redisutil.Prefix("rebuy-go-sdk-example")
		redisClient = redis.NewClient(&redis.Options{
			Addr: r.redisAddress,
		})
	)

	assetFSSub, err := fs.Sub(assetFS, "assets")
	cmdutil.Must(err)

	s := &Server{
		RedisClient: redisClient,
		RedisPrefix: redisPrefix,

		AssetFS: assetFSSub,
	}
	cmdutil.Must(s.Run(ctx))
}

// Dev initializes the server with local settings and starts mock-server where
// possible.
func (r *Runner) Dev(ctx context.Context, cmd *cobra.Command, args []string) {
	// Using miniredis instead of a real one, because that makes the local
	// environment requirements easier.
	redisFake, err := miniredis.Run()
	cmdutil.Must(err)

	var (
		redisPrefix = redisutil.Prefix("rebuy-go-sdk-example")
		redisClient = redis.NewClient(&redis.Options{
			Addr: redisFake.Addr(),
		})
	)

	s := &Server{
		RedisClient: redisClient,
		RedisPrefix: redisPrefix,

		// Reading directly from disk on dev mode, to be able to refresh the
		// browser without having to restart the server.
		AssetFS: os.DirFS("cmd/assets"),
	}
	cmdutil.Must(s.Run(ctx))
}
