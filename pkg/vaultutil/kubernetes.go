package vaultutil

import (
	"os"
	"path"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

const (
	KubernetesTokenPath      = "/run/secrets/kubernetes.io/serviceaccount/token"
	KubernetesAuthMountPoint = "kubernetes"
)

func KubernetesToken(client *api.Client, role string) (*api.Secret, error) {
	jwt, err := os.ReadFile(KubernetesTokenPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var (
		loc = path.Join("auth", KubernetesAuthMountPoint, "login")
		cfg = map[string]any{
			"role": role,
			"jwt":  string(jwt),
		}
	)

	secret, err := client.Logical().Write(loc, cfg)
	return secret, errors.WithStack(err)
}
