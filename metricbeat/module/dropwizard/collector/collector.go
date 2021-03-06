package collector

import (
	"encoding/json"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "metrics_path",
	}.Build()
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data

func init() {
	if err := mb.Registry.AddMetricSet("dropwizard", "collector", New, hostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	http      *helper.HTTP
	namespace string
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Beta("The dropwizard collector metricset is beta")
	config := struct {
		Namespace string `config:"namespace" validate:"required"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          helper.NewHTTP(base),
		namespace:     config.Namespace,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	body, err := m.http.FetchContent()
	if err != nil {
		return nil, err
	}
	dw := map[string]interface{}{}

	d := json.NewDecoder(strings.NewReader(string(body)))
	d.UseNumber()

	err = d.Decode(&dw)
	if err != nil {
		return nil, err
	}

	eventList := eventMapping(dw)

	// Converts hash list to slice
	events := []common.MapStr{}
	for _, e := range eventList {
		e["_namespace"] = m.namespace
		events = append(events, e)
	}

	return events, err

}
