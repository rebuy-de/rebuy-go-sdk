package instutil

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
)

func init() {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "rebuy",
		Name:      "toolstack",
	}, []string{
		"toolstack",
		"version",
	})
	prometheus.MustRegister(gauge)

	major := strings.SplitN(cmdutil.SDKVersion, ".", 2)[0]

	gauge.WithLabelValues(
		"golang",
		cmdutil.GoVersion,
	).Set(1)

	gauge.WithLabelValues(
		"rebuy-go-sdk",
		cmdutil.SDKVersion,
	).Set(1)
}
