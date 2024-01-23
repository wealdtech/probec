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

package mock

import (
	"context"

	submitter "github.com/wealdtech/probec/services/submitter"
)

// Service is a mock submitter.
type service struct{}

// New creates a new mock submitter.
func New() submitter.Service {
	return &service{}
}

// SubmitBlockDelay submits a block delay data point.
func (s *service) SubmitBlockDelay(ctx context.Context, body string) {}

// SubmitHeadDelay submits a head delay data point.
func (s *service) SubmitHeadDelay(ctx context.Context, body string) {}

// SubmitAggregateAttestation submits an aggregate attestation data point.
func (s *service) SubmitAggregateAttestation(ctx context.Context, body string) {}

// SubmitAttestationSummary submits a summary of attestation data points.
func (s *service) SubmitAttestationSummary(ctx context.Context, body string) {}
