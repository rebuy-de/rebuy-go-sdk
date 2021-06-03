package kubeutil

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	FlagKubeconfig = "kubeconfig"
	EnvKubeconfig  = "KUBECONFIG"
)

type Params struct {
	kubeconfig string
}

func (p *Params) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&p.kubeconfig, "kubeconfig", "",
		fmt.Sprintf("Path to the kubeconfig file to use for Kubernetes requests ($%s)", EnvKubeconfig))

	return nil
}

func (p *Params) Config() (*rest.Config, error) {
	if p.kubeconfig == "" {
		p.kubeconfig = os.Getenv(EnvKubeconfig)
	}

	config, err := clientcmd.BuildConfigFromFlags("", p.kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kubernetes config")
	}

	return config, nil
}

func (p *Params) Client() (kubernetes.Interface, error) {
	config, err := p.Config()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize kubernetes client")
	}

	return client, nil
}
