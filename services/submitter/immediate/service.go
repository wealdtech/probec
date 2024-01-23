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
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	zerologger "github.com/rs/zerolog/log"
)

// Service is a fee recipient provider service.
type Service struct {
	log      zerolog.Logger
	baseURLs []string
}

// New creates a new fee recipient provider service.
func New(ctx context.Context, params ...Parameter) (*Service, error) {
	parameters, err := parseAndCheckParameters(params...)
	if err != nil {
		return nil, errors.Wrap(err, "problem with parameters")
	}

	// Set logging.
	log := zerologger.With().Str("service", "submitter").Str("impl", "immediate").Logger()
	if parameters.logLevel != log.GetLevel() {
		log = log.Level(parameters.logLevel)
	}

	if err := registerMetrics(ctx, parameters.monitor); err != nil {
		return nil, errors.New("failed to register metrics")
	}

	baseURLs := make([]string, len(parameters.baseURLs))
	for i := range parameters.baseURLs {
		baseURL, err := url.Parse(parameters.baseURLs[i])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid base URL %s", parameters.baseURLs[i])
		}
		baseURLs[i] = strings.TrimSuffix(baseURL.String(), "/")
	}

	s := &Service{
		log:      log,
		baseURLs: baseURLs,
	}

	return s, nil
}
