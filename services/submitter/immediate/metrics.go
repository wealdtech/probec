// Copyright © 2022 Weald Technology Trading.
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

package immediate

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/wealdtech/probec/services/metrics"
)

var (
	submitterCounter *prometheus.CounterVec
	submitterTimer   *prometheus.HistogramVec
)

func registerMetrics(ctx context.Context, monitor metrics.Service) error {
	if submitterCounter != nil {
		// Already registered.
		return nil
	}
	if monitor == nil {
		// No monitor.
		return nil
	}
	if monitor.Presenter() == "prometheus" {
		return registerPrometheusMetrics(ctx)
	}

	return nil
}

func registerPrometheusMetrics(_ context.Context) error {
	submitterCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "probec",
		Subsystem: "submitter",
		Name:      "requests_total",
		Help:      "Total number of requests submitted",
	}, []string{"operation", "result"})
	if err := prometheus.Register(submitterCounter); err != nil {
		return err
	}
	submitterTimer = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "probec",
		Subsystem: "submitter",
		Name:      "duration_seconds",
		Help:      "The time spent submitting data.",
		Buckets: []float64{
			0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0,
			1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9, 2.0,
			2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 2.9, 3.0,
			3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9, 4.0,
		},
	}, []string{"operation"})

	return prometheus.Register(submitterTimer)
}

// monitorSubmission is called when a submission has been made.
func monitorSubmission(operation string, succeeded bool, delay time.Duration) {
	if submitterCounter == nil {
		return
	}

	if succeeded {
		submitterCounter.WithLabelValues(operation, "succeeded").Inc()
		submitterTimer.WithLabelValues(operation).Observe(delay.Seconds())
	} else {
		submitterCounter.WithLabelValues(operation, "failed").Inc()
	}
}
