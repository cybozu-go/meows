package metrics

import (
	"context"
	"errors"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
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

type Counter struct {
	Label map[string]string
	Value float64
}

// FetchCounter fetches a counter metrics. This is a helper function for tests.
func FetchCounter(ctx context.Context, url, name string) ([]*Counter, error) {
	family, err := fetchFamily(ctx, url, name, dto.MetricType_COUNTER)
	if err != nil {
		return nil, err
	}

	var ret []*Counter
	for _, m := range family.GetMetric() {
		ret = append(ret, &Counter{
			Label: labelMap(m),
			Value: m.GetCounter().GetValue(),
		})
	}
	return ret, nil
}

type Histogram struct {
	Label       map[string]string
	SampleCount uint64
	SampleSum   float64
}

// FetchHistogram fetches a histogram metrics. This is a helper function for tests.
func FetchHistogram(ctx context.Context, url, name string) ([]*Histogram, error) {
	family, err := fetchFamily(ctx, url, name, dto.MetricType_HISTOGRAM)
	if err != nil {
		return nil, err
	}

	var ret []*Histogram
	for _, m := range family.GetMetric() {
		ret = append(ret, &Histogram{
			Label:       labelMap(m),
			SampleCount: m.GetHistogram().GetSampleCount(),
			SampleSum:   m.GetHistogram().GetSampleSum(),
		})
	}
	return ret, nil
}

func fetchFamily(ctx context.Context, url, name string, metricType dto.MetricType) (*dto.MetricFamily, error) {
	families, err := fetchMetricsFamily(ctx, url)
	if err != nil {
		return nil, err
	}

	family := families[name]
	if family == nil {
		return nil, ErrNotExist
	}
	if family.GetType() != metricType {
		return nil, ErrWrongType
	}
	return family, nil
}

func labelMap(m *dto.Metric) map[string]string {
	label := map[string]string{}
	for _, l := range m.GetLabel() {
		label[l.GetName()] = l.GetValue()
	}
	return label
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

	parser := expfmt.NewTextParser(model.UTF8Validation)
	return parser.TextToMetricFamilies(resp.Body)
}
