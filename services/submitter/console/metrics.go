// Copyright © 2023 Weald Technology Trading.
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

package console

import (
	"context"

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
	}, []string{"operation", "result"})
	if err := prometheus.Register(submitterCounter); err != nil {
		return err
	}
	return prometheus.Register(submitterTimer)
}

// monitorSubmission is called when a submission has been made.
func monitorSubmission(operation string) {
	if submitterCounter == nil {
		return
	}

	submitterCounter.WithLabelValues(operation, "succeeded").Inc()
}
