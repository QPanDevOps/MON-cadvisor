// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
	v2 "github.com/google/cadvisor/info/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	clock "k8s.io/utils/clock/testing"
)

var now = clock.NewFakeClock(time.Unix(1395066363, 0))

func TestPrometheusCollector(t *testing.T) {
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics, now, v2.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics")
}

func testPrometheusCollector(t *testing.T, gatherer prometheus.Gatherer, metricsFile string) {
	wantMetrics, err := os.Open(metricsFile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", metricsFile)
	}

	err = testutil.GatherAndCompare(gatherer, wantMetrics)
	if err != nil {
		t.Fatalf("Metric comparison failed: %s", err)
	}
}

func TestPrometheusCollector_scrapeFailure(t *testing.T) {
	provider := &erroringSubcontainersInfoProvider{
		successfulProvider: testSubcontainersInfoProvider{},
		shouldFail:         true,
	}

	c := NewPrometheusCollector(provider, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics, now, v2.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics_failure")

	provider.shouldFail = false

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics")
}

func TestNewPrometheusCollectorWithPerf(t *testing.T) {
	c := NewPrometheusCollector(&mockInfoProvider{}, mockLabelFunc, container.MetricSet{container.PerfMetrics: struct{}{}}, now, v2.RequestOptions{})
	assert.Len(t, c.containerMetrics, 5)
	names := []string{}
	for _, m := range c.containerMetrics {
		names = append(names, m.name)
	}
	assert.Contains(t, names, "container_last_seen")
	assert.Contains(t, names, "container_perf_events_total")
	assert.Contains(t, names, "container_perf_events_scaling_ratio")
	assert.Contains(t, names, "container_perf_uncore_events_total")
	assert.Contains(t, names, "container_perf_uncore_events_scaling_ratio")
}

func TestNewPrometheusCollectorWithRequestOptions(t *testing.T) {
	p := mockInfoProvider{}
	opts := v2.RequestOptions{
		IdType: "docker",
	}
	c := NewPrometheusCollector(&p, mockLabelFunc, container.AllMetrics, now, opts)
	ch := make(chan prometheus.Metric, 10)
	c.Collect(ch)
	assert.Equal(t, p.options, opts)
}

type mockInfoProvider struct {
	options v2.RequestOptions
}

func (m *mockInfoProvider) GetRequestedContainersInfo(containerName string, options v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	m.options = options
	return map[string]*info.ContainerInfo{}, nil
}

func (m *mockInfoProvider) GetVersionInfo() (*info.VersionInfo, error) {
	return nil, errors.New("not supported")
}

func (m *mockInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	return nil, errors.New("not supported")
}

func mockLabelFunc(*info.ContainerInfo) map[string]string {
	return map[string]string{}
}
