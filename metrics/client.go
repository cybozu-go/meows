package metrics

import (
	"context"
	"errors"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var (
	ErrNotExist  = errors.New("metrics is not exist")
	ErrWrongType = errors.New("metrics type is wrong")
)

type Gauge struct {
	Label map[string]string
	Value float64
}

// FetchGauge fetches a gauge metrics. This is a helper function for tests.
func FetchGauge(ctx context.Context, url, name string) ([]*Gauge, error) {
	families, err := fetchMetricsFamily(ctx, url)
	if err != nil {
		return nil, err
	}

	family := families[name]
	if family == nil {
		return nil, ErrNotExist
	}
	if family.GetType() != dto.MetricType_GAUGE {
		return nil, ErrWrongType
	}

	var ret []*Gauge
	for _, m := range family.GetMetric() {
		label := map[string]string{}
		for _, l := range m.GetLabel() {
			label[l.GetName()] = l.GetValue()
		}
		ret = append(ret, &Gauge{
			Label: label,
			Value: m.GetGauge().GetValue(),
		})
	}
	return ret, nil
}

func fetchMetricsFamily(ctx context.Context, url string) (map[string]*dto.MetricFamily, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return (&expfmt.TextParser{}).TextToMetricFamilies(resp.Body)
}
