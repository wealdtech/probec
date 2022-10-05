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

package events

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	consensusclient "github.com/attestantio/go-eth2-client"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	bitfield "github.com/prysmaticlabs/go-bitfield"
	"github.com/rs/zerolog"
	zerologger "github.com/rs/zerolog/log"
	"github.com/wealdtech/chaind/services/chaintime"
	"github.com/wealdtech/probec/services/submitter"
)

// attestationSummary provides a summary of attestations for a given vote.
type attestationSummary struct {
	committee       phase0.CommitteeIndex
	beaconBlockRoot phase0.Root
	sourceRoot      phase0.Root
	targetRoot      phase0.Root
	buckets         map[string]*[120]bitfield.Bitlist
}

// Service is an attestations tarcker service.
type Service struct {
	chainTime            chaintime.Service
	submitter            submitter.Service
	attestationsMu       sync.Mutex
	attestationSummaries map[phase0.Slot]map[string]*attestationSummary
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
	log = zerologger.With().Str("service", "attestations").Str("impl", "events").Logger()
	if parameters.logLevel != log.GetLevel() {
		log = log.Level(parameters.logLevel)
	}

	if err := registerMetrics(ctx, parameters.monitor); err != nil {
		return nil, errors.New("failed to register metrics")
	}

	s := &Service{
		chainTime:            parameters.chainTime,
		submitter:            parameters.submitter,
		attestationSummaries: make(map[phase0.Slot]map[string]*attestationSummary),
	}

	for name, eventsProvider := range parameters.eventsProviders {
		if err := s.monitorEvents(ctx, name, eventsProvider); err != nil {
			return nil, err

		}
	}
	return s, nil
}

func (s *Service) monitorEvents(ctx context.Context,
	name string,
	eventsProvider consensusclient.EventsProvider,
) error {
	if err := eventsProvider.Events(ctx, []string{"attestation"}, func(event *apiv1.Event) {
		data := event.Data.(*phase0.Attestation)
		delay := time.Since(s.chainTime.StartOfSlot(data.Data.Slot))
		if delay.Seconds() < 0 || delay.Seconds() > 12 {
			log.Debug().Stringer("delay", delay).Msg("Delay out of range, ignoring")
			return
		}
		monitorEventProcessed(delay)

		// We treat attestations differently depending on if they are individual or aggregate.
		validators := data.AggregationBits.Count()
		if validators == 1 {
			if err := s.handleAttestation(ctx, name, data, delay); err != nil {
				log.Error().Err(err).Msg("Failed to handle attestation")
			}
		} else {
			if err := s.handleAggregateAttestation(ctx, name, data, delay); err != nil {
				log.Error().Err(err).Msg("Failed to handle aggregate attestation")
			}
		}
	}); err != nil {
		return errors.Wrap(err, "failed to create events provider")
	}

	return nil
}

func (s *Service) handleAttestation(ctx context.Context,
	name string,
	attestation *phase0.Attestation,
	delay time.Duration,
) error {
	bucket := delay.Milliseconds() % 100
	if bucket < 0 || bucket > 119 {
		log.Debug().Int64("bucket", bucket).Msg("Bucket out of range; ignoring")
		return nil
	}

	key := fmt.Sprintf("%d:%x:%x:%x", attestation.Data.Index, attestation.Data.BeaconBlockRoot, attestation.Data.Source.Root, attestation.Data.Target.Root)
	s.attestationsMu.Lock()
	slotSummaries, exists := s.attestationSummaries[attestation.Data.Slot]
	if !exists {
		slotSummaries = make(map[string]*attestationSummary)
		s.attestationSummaries[attestation.Data.Slot] = slotSummaries
	}
	summary, exists := slotSummaries[key]
	if !exists {
		summary = &attestationSummary{
			committee:       attestation.Data.Index,
			beaconBlockRoot: attestation.Data.BeaconBlockRoot,
			sourceRoot:      attestation.Data.Source.Root,
			targetRoot:      attestation.Data.Target.Root,
			buckets:         map[string]*[120]bitfield.Bitlist{},
		}
		slotSummaries[key] = summary
	}
	buckets, exists := summary.buckets[name]
	if !exists {
		buckets = &[120]bitfield.Bitlist{}
		summary.buckets[name] = buckets
	}
	if buckets[bucket] == nil {
		buckets[bucket] = attestation.AggregationBits
	} else {
		var err error
		buckets[bucket], err = buckets[bucket].Or(attestation.AggregationBits)
		if err != nil {
			s.attestationsMu.Unlock()
			log.Error().Err(err).Msg("Failed to aggregate attestations")
			return nil
		}
	}

	lastSlotSummaries, exists := s.attestationSummaries[attestation.Data.Slot-1]
	if !exists {
		s.attestationsMu.Unlock()
		return nil
	}

	delete(s.attestationSummaries, attestation.Data.Slot-1)
	s.attestationsMu.Unlock()

	// Build and send the data.
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf(`{"method":"attestation event","slot":"%d","attestations":[`, attestation.Data.Slot-1))
	firstSummary := true
	for _, summary := range lastSlotSummaries {
		if firstSummary {
			firstSummary = false
		} else {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf(`{"committee_index":"%d","beacon_block_root":"%#x","source_root":"%#x","target_root":"%#x","buckets":`, summary.committee, summary.beaconBlockRoot, summary.sourceRoot, summary.targetRoot))
		builder.WriteString(`{`)
		firstSource := true
		for source, sourceBuckets := range summary.buckets {
			if firstSource {
				firstSource = false
			} else {
				builder.WriteString(",")
			}
			builder.WriteString(fmt.Sprintf(`"%s":[`, source))
			firstBucket := true
			for _, sourceBucket := range sourceBuckets {
				if firstBucket {
					firstBucket = false
				} else {
					builder.WriteString(",")
				}
				builder.WriteString(fmt.Sprintf(`"%#x"`, sourceBucket))
			}
			builder.WriteString(`]`)
		}
		builder.WriteString(`}}`)
	}
	builder.WriteString("]}")
	log.Trace().RawJSON("data", []byte(builder.String())).Msg("Attestation summary")

	if err := s.submitter.SubmitAttestationSummary(ctx, builder.String()); err != nil {
		return errors.Wrap(err, "failed to submit attestation summary")
	}

	return nil
}

func (s *Service) handleAggregateAttestation(ctx context.Context,
	name string,
	attestation *phase0.Attestation,
	delay time.Duration,
) error {
	// Build and send the data.
	body := fmt.Sprintf(`{"source":"%s","method":"attestation event","slot":"%d","committee_index":"%d","beacon_block_root":"%#x","source_root":"%#x","target_root":"%#x","aggregation_bits":"%#x","delay_ms":"%d"}`,
		name,
		attestation.Data.Slot,
		attestation.Data.Index,
		attestation.Data.BeaconBlockRoot,
		attestation.Data.Source.Root,
		attestation.Data.Target.Root,
		attestation.AggregationBits,
		int(delay.Milliseconds()),
	)
	log.Trace().RawJSON("data", []byte(body)).Msg("Aggregate attestation")
	if err := s.submitter.SubmitAggregateAttestation(ctx, body); err != nil {
		return errors.Wrap(err, "failed to submit aggregate attestation")
	}
	return nil
}
