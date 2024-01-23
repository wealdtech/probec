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

package immediate

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SubmitAttestationSummary submits a summary of attestation data points.
func (s *Service) SubmitAttestationSummary(ctx context.Context, body string) {
	for _, baseURL := range s.baseURLs {
		go s.submitAttestationSummary(ctx, body, baseURL)
	}
}

func (s *Service) submitAttestationSummary(ctx context.Context, body string, baseURL string) {
	started := time.Now()

	url := fmt.Sprintf("%s/v1/attestationsummary", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		monitorSubmission("attestation summary", false, time.Since(started))
		s.log.Error().Err(err).Msg("Failed to create attestation summary request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		monitorSubmission("attestation summary", false, time.Since(started))
		s.log.Error().Err(err).Msg("Failed to send attestation summary request")
	}

	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			monitorSubmission("attestation summary", false, time.Since(started))
			return
		}
	}

	monitorSubmission("attestation summary", true, time.Since(started))
}
