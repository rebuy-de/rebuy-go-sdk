package cmd

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v5/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v5/pkg/podutil"
	"github.com/rebuy-de/rebuy-go-sdk/v5/pkg/redisutil"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//go:embed assets/*
var assetFS embed.FS

//go:embed templates
var templateFS embed.FS

// NewRootCommand initializes the cobra.Command with support of the cmdutil
// package.
func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"full", "rebuy-go-sdk-full-example",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithLogToGraylog(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		cmdutil.WithSubCommand(cmdutil.New(
			"daemon", "Run the application",
			cmdutil.WithRunner(new(DaemonRunner)),
		)),

		cmdutil.WithSubCommand(cmdutil.New(
			"dev", "Run the application in dev mode for local development",
			cmdutil.WithRunner(new(DevRunner)),
		)),
	)
}

// DaemonRunner bootstraps the application for production. It defines the
// related flags and calls the actual server code.
type DaemonRunner struct {
	redisAddress string
}

// Bind implements the cmdutil.Runner interface and defines command line flags.
func (r *DaemonRunner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.redisAddress, "redis-address", "",
		`Address of the Redis instance.`)
	return nil
}

// Daemon initializes the server with production-ready settings.
func (r *DaemonRunner) Run(ctx context.Context) error {
	var (
		redisPrefix = redisutil.Prefix("rebuy-go-sdk-example")
		redisClient = redis.NewClient(&redis.Options{
			Addr: r.redisAddress,
		})
	)

	assetFSSub, err := fs.Sub(assetFS, "assets")
	if err != nil {
		return fmt.Errorf("open assets dir: %w", err)
	}

	templateFSSub, err := fs.Sub(templateFS, "templates")
	if err != nil {
		return fmt.Errorf("open templates dir: %w", err)
	}

	s := &Server{
		RedisClient: redisClient,
		RedisPrefix: redisPrefix,

		AssetFS:    assetFSSub,
		TemplateFS: templateFSSub,
	}

	return s.Run(ctx)
}

// DevRunner bootstraps the application for local development. It defines the
// related flags and calls the actual server code.
type DevRunner struct {
	redisAddress string
}

// Bind implements the cmdutil.Runner interface and defines command line flags.
func (r *DevRunner) Bind(cmd *cobra.Command) error {
	return nil
}

// Run initializes the server with local settings and starts mock-server where
// possible.
func (r *DevRunner) Run(ctx context.Context) error {
	podman, err := podutil.New(podutil.UserSocketPath())
	if err != nil {
		return fmt.Errorf("init podman: %w", err)
	}

	keydbContainer, err := podutil.StartDevcontainer(ctx, podman, "boombot-dev-keydb",
		"docker.io/eqalpha/keydb:latest")
	if err != nil {
		return fmt.Errorf("start keydb: %w", err)
	}

	var (
		redisPrefix = redisutil.Prefix("rebuy-go-sdk-example")
		redisClient = redis.NewClient(&redis.Options{
			Addr: keydbContainer.TCPHostPort(6379),
		})
	)

	s := &Server{
		RedisClient: redisClient,
		RedisPrefix: redisPrefix,

		// Reading directly from disk on dev mode, to be able to refresh the
		// browser without having to restart the server.
		AssetFS:    os.DirFS("cmd/assets"),
		TemplateFS: os.DirFS("cmd/templates"),
	}

	return s.Run(ctx)
}
