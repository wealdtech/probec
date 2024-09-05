// Copyright © 2022, 2024 Weald Technology Trading.
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

package events

import (
	"errors"

	consensusclient "github.com/attestantio/go-eth2-client"
	"github.com/rs/zerolog"
	"github.com/wealdtech/probec/services/chaintime"
	"github.com/wealdtech/probec/services/metrics"
	nullmetrics "github.com/wealdtech/probec/services/metrics/null"
	"github.com/wealdtech/probec/services/submitter"
)

type parameters struct {
	logLevel             zerolog.Level
	monitor              metrics.Service
	chainTime            chaintime.Service
	eventsProviders      map[string]consensusclient.EventsProvider
	nodeVersionProviders map[string]consensusclient.NodeVersionProvider
	submitter            submitter.Service
}

// Parameter is the interface for service parameters.
type Parameter interface {
	apply(*parameters)
}

type parameterFunc func(*parameters)

func (f parameterFunc) apply(p *parameters) {
	f(p)
}

// WithLogLevel sets the log level for the module.
func WithLogLevel(logLevel zerolog.Level) Parameter {
	return parameterFunc(func(p *parameters) {
		p.logLevel = logLevel
	})
}

// WithMonitor sets the monitor for the module.
func WithMonitor(monitor metrics.Service) Parameter {
	return parameterFunc(func(p *parameters) {
		p.monitor = monitor
	})
}

// WithChainTime sets the chain time service for this module.
func WithChainTime(service chaintime.Service) Parameter {
	return parameterFunc(func(p *parameters) {
		p.chainTime = service
	})
}

// WithEventsProviders sets the events providers for this module.
func WithEventsProviders(providers map[string]consensusclient.EventsProvider) Parameter {
	return parameterFunc(func(p *parameters) {
		p.eventsProviders = providers
	})
}

// WithNodeVersionProviders sets the node version providers for this module.
func WithNodeVersionProviders(providers map[string]consensusclient.NodeVersionProvider) Parameter {
	return parameterFunc(func(p *parameters) {
		p.nodeVersionProviders = providers
	})
}

// WithSubmitter sets the submitter for this module.
func WithSubmitter(submitter submitter.Service) Parameter {
	return parameterFunc(func(p *parameters) {
		p.submitter = submitter
	})
}

// parseAndCheckParameters parses and checks parameters to ensure that mandatory parameters are present and correct.
func parseAndCheckParameters(params ...Parameter) (*parameters, error) {
	parameters := parameters{
		logLevel: zerolog.GlobalLevel(),
		monitor:  nullmetrics.New(),
	}
	for _, p := range params {
		if params != nil {
			p.apply(&parameters)
		}
	}

	if parameters.monitor == nil {
		return nil, errors.New("monitor not supplied")
	}
	if parameters.chainTime == nil {
		return nil, errors.New("chain time service not supplied")
	}
	if len(parameters.eventsProviders) == 0 {
		return nil, errors.New("events providers not supplied")
	}
	if len(parameters.nodeVersionProviders) == 0 {
		return nil, errors.New("node version providers not supplied")
	}
	if parameters.submitter == nil {
		return nil, errors.New("submitter not supplied")
	}

	return &parameters, nil
}
