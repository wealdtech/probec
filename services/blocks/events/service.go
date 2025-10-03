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
	"context"
	"fmt"
	"time"

	consensusclient "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	zerologger "github.com/rs/zerolog/log"
	"github.com/wealdtech/probec/services/chaintime"
	"github.com/wealdtech/probec/services/submitter"
)

// Service is a fee recipient provider service.
type Service struct {
	chainTime chaintime.Service
	submitter submitter.Service
}

// module-wide log.
var log zerolog.Logger

// New creates a new fee recipient provider service.
func New(ctx context.Context, params ...Parameter) (*Service, error) {
	parameters, err := parseAndCheckParameters(params...)
	if err != nil {
		return nil, errors.Wrap(err, "problem with parameters")
	}

	// Set logging.
	log = zerologger.With().Str("service", "blocks").Str("impl", "events").Logger()
	if parameters.logLevel != log.GetLevel() {
		log = log.Level(parameters.logLevel)
	}

	if err := registerMetrics(ctx, parameters.monitor); err != nil {
		return nil, errors.New("failed to register metrics")
	}

	s := &Service{
		chainTime: parameters.chainTime,
		submitter: parameters.submitter,
	}

	for address, eventsProvider := range parameters.eventsProviders {
		if err := s.monitorEvents(ctx, eventsProvider, parameters.nodeVersionProviders[address]); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Service) monitorEvents(ctx context.Context,
	eventsProvider consensusclient.EventsProvider,
	nodeVersionProvider consensusclient.NodeVersionProvider,
) error {
	if err := eventsProvider.Events(ctx, &api.EventsOpts{
		BlockHandler: func(ctx context.Context, event *apiv1.BlockEvent) {
			delay := time.Since(s.chainTime.StartOfSlot(event.Slot))

			// Ensure the node is synced.
			syncingProvider, ok := eventsProvider.(consensusclient.NodeSyncingProvider)
			if !ok {
				log.Error().Msg("Node syncing provider not supported")
				return
			}
			syncingResponse, err := syncingProvider.NodeSyncing(ctx, &api.NodeSyncingOpts{})
			if err != nil {
				log.Error().Err(err).Msg("Failed to ascertain if node is syncing")
				return
			}
			if syncingResponse.Data.IsSyncing {
				log.Debug().Msg("Node is syncing, not sending information")
			}

			monitorEventProcessed(delay)

			nodeVersionResponse, err := nodeVersionProvider.NodeVersion(ctx, &api.NodeVersionOpts{})
			if err != nil {
				log.Error().Err(err).Msg("Failed to obtain node version")
				return
			}

			// Build and send the data.
			body := fmt.Sprintf(
				`{"source":"%s","method":"block event","slot":"%d","delay_ms":"%d"}`,
				nodeVersionResponse.Data,
				event.Slot,
				int(delay.Milliseconds()),
			)
			s.submitter.SubmitBlockDelay(ctx, body)
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create events provider")
	}

	return nil
}
