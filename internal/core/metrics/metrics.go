package metrics

import (
	"strings"

	"git.ptb.bet/public-group/shared/v2/pkg/metrics"
	"github.com/alexbro4u/gotemplate/internal/config"

	"github.com/prometheus/client_golang/prometheus"
)

type Factory struct {
	*metrics.Factory

	constLabels prometheus.Labels

	HTTPMetrics *metrics.HTTPMetrics
}

func New(cfg *config.Config) (*Factory, error) {
	constLabels := parseConstLabels(cfg.Metrics.ConstLabels)

	opts := []metrics.Option{
		metrics.WithNamespace(cfg.Metrics.Namespace),
		metrics.WithSubsystem(cfg.Metrics.Subsystem),
		metrics.WithConstLabels(constLabels),
	}

	factory, err := metrics.New(opts...)
	if err != nil {
		return nil, err
	}

	httpMetrics, err := factory.HTTPMetrics()
	if err != nil {
		return nil, err
	}

	return &Factory{
		Factory:     factory,
		constLabels: constLabels,
		HTTPMetrics: httpMetrics,
	}, nil
}

func (f *Factory) EnvLabel() string {
	if value, ok := f.constLabels["env"]; ok {
		return value
	}

	return ""
}

func parseConstLabels(raw string) prometheus.Labels {
	result := make(prometheus.Labels)
	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if key != "" {
				result[key] = strings.TrimSpace(parts[1])
			}
		}
	}
	if len(result) == 0 {
		result["app"] = "gotemplate"
	}
	return result
}
