package vaultutil

import (
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
)

func DecodeSecret[T any](secret *api.Secret) (T, error) {
	return DecodeSecretWithPrefix[T](secret, "")
}

func DecodeSecretWithPrefix[T any](secret *api.Secret, prefix string) (T, error) {
	var result T
	config := &mapstructure.DecoderConfig{
		Result:  &result,
		TagName: "vault",
		MatchName: func(mapKey, fieldName string) bool {
			return strings.EqualFold(mapKey, prefix+fieldName)
		},
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return result, err
	}

	err = decoder.Decode(secret.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

type GitHubSecrets struct {
	AppID          int64  `vault:"github-app-id"`
	InstallationID int64  `vault:"github-installation-id"`
	PrivateKey     string `vault:"github-private-key"`
}

type SlackSecrets struct {
	VerificationToken string `vault:"slack-verification-token"`
	SigningSecret     string `vault:"slack-signing-secret"`
	Token             string `vault:"slack-token"`
	AppToken          string `vault:"slack-app-token"`
	Channel           string `vault:"slack-channel"`
}

type SlackTestSecrets struct {
	VerificationToken string `vault:"slack-test-verification-token"`
	SigningSecret     string `vault:"slack-test-signing-secret"`
	Token             string `vault:"slack-test-token"`
	AppToken          string `vault:"slack-test-app-token"`
	Channel           string `vault:"slack-test-channel"`
}

type OIDCSecrets struct {
	ClientID     string `vault:"oidc-client-id"`
	ClientSecret string `vault:"oidc-client-secret"`
}
