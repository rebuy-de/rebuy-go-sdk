package vaultutil

import (
	"context"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/logutil"
)

type awsCredentialsProvider struct {
	manager *Manager
}

func (p *awsCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	var value aws.Credentials
	value.Source = "vault"

	secret, err := p.manager.client.Logical().Read(path.Join(p.manager.params.AWSEnginePath, "creds", p.manager.params.AWSRole))
	if err != nil {
		return value, errors.WithStack(err)
	}

	logutil.Get(p.manager.ctx).
		WithField("secret-data", prettyPrintSecret(secret)).
		Debugf("created new AWS lease")

	var (
		duration   = time.Duration(secret.LeaseDuration) * time.Second
		expiration = time.Now().Add(duration)
		// The CredentialsCache layer is refreshing credentials
		// 5 minutes before expiry by default
		// https://github.com/aws/aws-sdk-go-v2/blob/v1.4.0/config/resolve_credentials.go#L252
	)
	value.CanExpire = true
	value.Expires = expiration

	// The blank identifier avoids throwing a panic, if the data is not a
	// string for some reason.
	value.AccessKeyID, _ = secret.Data["access_key"].(string)
	value.SecretAccessKey, _ = secret.Data["secret_key"].(string)
	value.SessionToken, _ = secret.Data["security_token"].(string)

	return value, nil
}
