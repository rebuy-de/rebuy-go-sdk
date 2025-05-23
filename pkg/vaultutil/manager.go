package vaultutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
)

const (
	RenewIntervalSeconds = 1800
)

type Manager struct {
	ctx    context.Context
	params Params
	client *api.Client
}

func Init(ctx context.Context, params Params) (*Manager, error) {
	ctx = logutil.Start(ctx, "vault-manager")

	conf := api.DefaultConfig()
	conf.Address = params.Address

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if params.Token == "" {
		secret, err := KubernetesToken(client, params.Role)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		client.SetToken(secret.Auth.ClientToken)
	} else {
		client.SetToken(params.Token)
	}

	secret, err := client.Auth().Token().RenewSelf(RenewIntervalSeconds)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	client.SetToken(secret.Auth.ClientToken)

	logutil.Get(ctx).Debug("got initial secret", "secret-data", prettyPrintSecret(secret))

	watcher, err := client.NewLifetimeWatcher(&api.LifetimeWatcherInput{
		Secret:      secret,
		RenewBuffer: 3,
		Increment:   RenewIntervalSeconds,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	go watcher.Start()

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for ctx.Err() == nil {
			select {
			case out := <-watcher.RenewCh():
				logutil.Get(ctx).Debug("renewed secret", "secret-data", prettyPrintSecret(out.Secret))

			case err := <-watcher.DoneCh():
				logutil.Get(ctx).Error("renewal stopped", "error", errors.WithStack(err))
				cancel()

			case <-ctx.Done():
				logutil.Get(ctx).Warn("renewal canceled")
				cancel()
			}
		}

		logutil.Get(ctx).Warn("shutting down vault manager", "error", errors.WithStack(err))
		watcher.Stop()
		err := client.Auth().Token().RevokeSelf("")
		if err != nil {
			logutil.Get(ctx).Error("revoking own token failed", "error", errors.WithStack(err))
		} else {
			logutil.Get(ctx).Debug("revoking own token succeeded", "error", errors.WithStack(err))
		}
	}()

	return &Manager{
		ctx:    ctx,
		params: params,
		client: client,
	}, nil
}

func (m *Manager) GetClient() *api.Client {
	return m.client
}

func (m *Manager) AWSConfig(ctx context.Context) (*aws.Config, error) {
	var (
		provider = m.AWSCredentialsProvider()
	)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(provider),
		config.WithRegion("eu-west-1"),
	)
	return &cfg, errors.WithStack(err)
}

func (m *Manager) AWSCredentialsProvider() aws.CredentialsProvider {
	return &awsCredentialsProvider{
		manager: m,
	}
}
