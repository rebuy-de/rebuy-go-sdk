package instutil

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rebuy-de/rebuy-go-sdk/v3/pkg/cmdutil"
)

func init() {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "rebuy",
		Name:      "toolstack",
	}, []string{
		"toolstack",
		"language",
		"language_version",
		"sdk",
		"sdk_version",
	})
	prometheus.MustRegister(gauge)

	major := strings.SplitN(cmdutil.SDKVersion, ".", 2)[0]

	gauge.WithLabelValues(
		fmt.Sprintf("golang.rebuy-go-sdk.%s", major),
		"golang",
		cmdutil.GoVersion,
		"rebuy-go-sdk",
		cmdutil.SDKVersion,
	).Set(1)
}
