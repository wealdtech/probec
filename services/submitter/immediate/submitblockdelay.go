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

// SubmitBlockDelay submits a block delay data point.
func (s *Service) SubmitBlockDelay(ctx context.Context, body string) {
	for _, baseURL := range s.baseURLs {
		go s.submitBlockDelay(ctx, body, baseURL)
	}
}

func (s *Service) submitBlockDelay(ctx context.Context, body string, baseURL string) {
	started := time.Now()

	url := fmt.Sprintf("%s/v1/blockdelay", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		monitorSubmission("block delay", false, time.Since(started))
		s.log.Error().Err(err).Msg("Failed to create block delay request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		monitorSubmission("block delay", false, time.Since(started))
		s.log.Error().Err(err).Msg("Failed to send block delay request")
	}

	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			monitorSubmission("block delay", false, time.Since(started))
			return
		}
	}

	monitorSubmission("block delay", true, time.Since(started))
}
