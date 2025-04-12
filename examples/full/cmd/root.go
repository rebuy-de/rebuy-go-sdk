package cmd

import (
	"context"
	"errors"

	"github.com/alicebob/miniredis/v2"
	"github.com/rebuy-de/rebuy-go-sdk/v8/examples/full/web"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/dig"
)

func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"full-example", "A full example app for the rebuy-go-sdk.",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithLogToGraylog(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		cmdutil.WithSubCommand(
			cmdutil.New(
				"daemon", "Run the application as daemon",
				cmdutil.WithRunner(new(DaemonRunner)),
			)),

		cmdutil.WithSubCommand(cmdutil.New(
			"dev", "Run the application in local dev mode",
			cmdutil.WithRunner(new(DevRunner)),
		)),
	)
}

type DaemonRunner struct {
	redisAddress string
	redisPrefix  string
}

func (r *DaemonRunner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.redisAddress, "redis-address", "",
		`Address of the Redis instance.`)
	cmd.PersistentFlags().StringVar(
		&r.redisPrefix, "redis-prefix", "",
		`Prefix for redis keys.`)

	return nil
}

func (r *DaemonRunner) Run(ctx context.Context) error {
	c := dig.New()

	err := errors.Join(
		c.Provide(web.ProdFS),
		c.Provide(func() webutil.AssetFS { return web.ProdFS() }),
		c.Provide(webutil.AssetDefaultProd),
		c.Provide(func() *redis.Client {
			return redis.NewClient(&redis.Options{
				Addr: r.redisAddress,
			})
		}),
	)
	if err != nil {
		return err
	}

	return RunServer(ctx, c)
}

const (
	SomeTeam = `team-name`
)

type DevRunner struct {
	redisAddress string
}

func (r *DevRunner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.redisAddress, "redis-address", "",
		`Address of the Redis instance.`)

	return nil
}

func (r *DevRunner) Run(ctx context.Context) error {
	c := dig.New()

	redisAddress := r.redisAddress
	if redisAddress == "" {
		redisServer, err := miniredis.Run()
		if err != nil {
			return err
		}
		redisAddress = redisServer.Addr()
	}

	err := errors.Join(
		c.Provide(web.DevFS),
		c.Provide(webutil.AssetDefaultDev),
		c.Provide(func() *redis.Client {
			return redis.NewClient(&redis.Options{
				Addr: redisAddress,
			})
		}),
		c.Provide(func() webutil.AuthMiddleware { return webutil.DevAuthMiddleware(SomeTeam) }),
	)
	if err != nil {
		return err
	}

	return RunServer(ctx, c)
}
