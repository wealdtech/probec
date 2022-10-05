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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// SubmitAggregateAttestation submits an aggregate attestation data point.
func (s *Service) SubmitAggregateAttestation(ctx context.Context, body string) error {
	started := time.Now()

	resp, err := http.Post(fmt.Sprintf("%s/v1/aggregateattestation", s.baseUrl), "application/json", strings.NewReader(body))
	if err != nil {
		monitorSubmission("aggreate attestation", false, time.Since(started))
		return errors.Wrap(err, "failed to post aggregate attestation")
	}
	if err := resp.Body.Close(); err != nil {
		monitorSubmission("aggregate attestation", false, time.Since(started))
		return errors.Wrap(err, "failed to close aggregate attestation response")
	}

	monitorSubmission("aggregate attestation", true, time.Since(started))
	return nil
}
