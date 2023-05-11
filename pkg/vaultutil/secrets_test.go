package vaultutil

import (
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

func TestDecodeSecret(t *testing.T) {
	t.Run("GitHub", func(t *testing.T) {
		want := GitHubSecrets{
			AppID:          42,
			InstallationID: 1337,
			PrivateKey:     "blubber",
		}
		have, err := DecodeSecret[GitHubSecrets](&api.Secret{
			Data: map[string]any{
				"github-app-id":          42,
				"github-installation-id": 1337,
				"github-private-key":     "blubber",
			},
		})

		require.NoError(t, err)
		require.Equal(t, have, want)
	})

	t.Run("OIDC", func(t *testing.T) {
		want := OIDCSecrets{
			ClientID:     "horst",
			ClientSecret: "password",
		}
		have, err := DecodeSecret[OIDCSecrets](&api.Secret{
			Data: map[string]any{
				"oidc-client-id":     "horst",
				"oidc-client-secret": "password",
			},
		})

		require.NoError(t, err)
		require.Equal(t, have, want)
	})

	t.Run("Slack", func(t *testing.T) {
		want := SlackSecrets{
			VerificationToken: "a",
			SigningSecret:     "b",
			Token:             "c",
			AppToken:          "d",
			Channel:           "e",
		}
		have, err := DecodeSecret[SlackSecrets](&api.Secret{
			Data: map[string]any{
				"slack-verification-token": "a",
				"slack-signing-secret":     "b",
				"slack-token":              "c",
				"slack-app-token":          "d",
				"slack-channel":            "e",
			},
		})

		require.NoError(t, err)
		require.Equal(t, have, want)
	})

	t.Run("SlackTest", func(t *testing.T) {
		want := SlackSecrets{
			VerificationToken: "a",
			SigningSecret:     "b",
			Token:             "c",
			AppToken:          "d",
			Channel:           "e",
		}
		have, err := DecodeSecretWithPrefix[SlackSecrets](&api.Secret{
			Data: map[string]any{
				"test-slack-verification-token": "a",
				"test-slack-signing-secret":     "b",
				"test-slack-token":              "c",
				"test-slack-app-token":          "d",
				"test-slack-channel":            "e",
			},
		}, "test-")

		require.NoError(t, err)
		require.Equal(t, have, want)
	})

	t.Run("Nested", func(t *testing.T) {
		type secrets struct {
			Token  string        `vault:"token"`
			Slack  SlackSecrets  `vault:",squash"`
			GitHub GitHubSecrets `vault:",squash"`
		}

		want := secrets{
			Token: "much-secretz",
			Slack: SlackSecrets{
				VerificationToken: "a",
				SigningSecret:     "b",
				Token:             "c",
				AppToken:          "d",
				Channel:           "e",
			},
			GitHub: GitHubSecrets{
				AppID:          42,
				InstallationID: 1337,
				PrivateKey:     "blubber",
			},
		}
		have, err := DecodeSecretWithPrefix[secrets](&api.Secret{
			Data: map[string]any{
				"token":                    "much-secretz",
				"slack-verification-token": "a",
				"slack-signing-secret":     "b",
				"slack-token":              "c",
				"slack-app-token":          "d",
				"slack-channel":            "e",
				"github-app-id":            42,
				"github-installation-id":   1337,
				"github-private-key":       "blubber",
			},
		}, "test-")

		require.NoError(t, err)
		require.Equal(t, have, want)
	})
}
