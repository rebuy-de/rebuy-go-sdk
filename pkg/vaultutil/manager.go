package vaultutil

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/logutil"
)

const (
	RenewIntervalSeconds = 1800

	RevokeTimeout = 5 * time.Second
)

type Manager struct {
	ctx    context.Context
	params Params
	client *api.Client

	cancel    context.CancelFunc
	done      chan struct{}
	revokeErr error
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

	return start(ctx, client, params, watcher), nil
}

func start(ctx context.Context, client *api.Client, params Params, watcher *api.LifetimeWatcher) *Manager {
	manageCtx, cancel := context.WithCancel(ctx)

	m := &Manager{
		ctx:    ctx,
		params: params,
		client: client,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		defer close(m.done)
		m.revokeErr = manageToken(manageCtx, client, params, watcher)
	}()

	return m
}

// Close stops the token renewal and blocks until the token got revoked. It
// should get deferred right after Init, otherwise the process might exit
// while the revocation request is still in flight, which Vault standby nodes
// log as forwarding errors.
//
// It is safe to call Close multiple times and also after the context passed
// to Init got cancelled.
func (m *Manager) Close() error {
	m.cancel()
	<-m.done
	return m.revokeErr
}

func manageToken(ctx context.Context, client *api.Client, params Params, watcher *api.LifetimeWatcher) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for ctx.Err() == nil {
		select {
		case out := <-watcher.RenewCh():
			logutil.Get(ctx).Debug("renewed secret", "secret-data", prettyPrintSecret(out.Secret))

		case err := <-watcher.DoneCh():
			logutil.Get(ctx).Error("renewal stopped", "error", err)
			cancel()

		case <-ctx.Done():
			logutil.Get(ctx).Warn("renewal canceled")
			cancel()
		}
	}

	logutil.Get(ctx).Warn("shutting down vault manager")
	watcher.Stop()

	if params.Token == "" {
		revokeCtx, revokeCancel := context.WithTimeout(context.WithoutCancel(ctx), RevokeTimeout)
		defer revokeCancel()

		err := client.Auth().Token().RevokeSelfWithContext(revokeCtx, "")
		if err != nil {
			logutil.Get(ctx).Error("revoking own token failed", "error", err)
			return errors.WithStack(err)
		}
		logutil.Get(ctx).Debug("revoking own token succeeded")
	}

	return nil
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
