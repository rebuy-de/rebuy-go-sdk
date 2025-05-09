package instutil

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
)

func init() {
	toolstack := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "rebuy",
		Name:      "toolstack",
	}, []string{
		"toolstack",
		"version",
	})
	prometheus.MustRegister(toolstack)

	toolstack.WithLabelValues(
		"golang",
		cmdutil.GoVersion,
	).Set(1)

	toolstack.WithLabelValues(
		"rebuy-go-sdk",
		cmdutil.SDKVersion,
	).Set(1)

	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "rebuy",
		Name:      "buildinfo",
	}, []string{
		"builddate",
	})
	prometheus.MustRegister(buildInfo)

	buildInfo.WithLabelValues(
		cmdutil.BuildDate,
	).Set(1)
}
