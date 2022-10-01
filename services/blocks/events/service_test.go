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

package events_test

import (
	"context"
	"testing"

	consensusclient "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/mock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	standardchaintime "github.com/wealdtech/chaind/services/chaintime/standard"
	"github.com/wealdtech/probec/services/blocks/events"
	mocksubmitter "github.com/wealdtech/probec/services/submitter/mock"
)

func TestService(t *testing.T) {
	ctx := context.Background()

	mockClient, err := mock.New(ctx)
	require.NoError(t, err)

	chainTime, err := standardchaintime.New(ctx,
		standardchaintime.WithGenesisTimeProvider(mockClient),
		standardchaintime.WithSpecProvider(mockClient),
		standardchaintime.WithForkScheduleProvider(mockClient),
	)
	require.NoError(t, err)

	submitter := mocksubmitter.New()

	tests := []struct {
		name   string
		params []events.Parameter
		err    string
	}{
		{
			name: "MonitorMissing",
			params: []events.Parameter{
				events.WithLogLevel(zerolog.Disabled),
				events.WithMonitor(nil),
				events.WithChainTime(chainTime),
				events.WithEventsProviders(map[string]consensusclient.EventsProvider{
					"test": mockClient,
				}),
				events.WithSubmitter(submitter),
			},
			err: "problem with parameters: monitor not supplied",
		},
		{
			name: "ChainTimeMissing",
			params: []events.Parameter{
				events.WithLogLevel(zerolog.Disabled),
				events.WithEventsProviders(map[string]consensusclient.EventsProvider{
					"test": mockClient,
				}),
				events.WithSubmitter(submitter),
			},
			err: "problem with parameters: chain time service not supplied",
		},
		{
			name: "EventsProvidersEmpty",
			params: []events.Parameter{
				events.WithLogLevel(zerolog.Disabled),
				events.WithChainTime(chainTime),
				events.WithEventsProviders(map[string]consensusclient.EventsProvider{}),
				events.WithSubmitter(submitter),
			},
			err: "problem with parameters: events providers not supplied",
		},
		{
			name: "SubmitterMissing",
			params: []events.Parameter{
				events.WithLogLevel(zerolog.Disabled),
				events.WithChainTime(chainTime),
				events.WithEventsProviders(map[string]consensusclient.EventsProvider{
					"test": mockClient,
				}),
			},
			err: "problem with parameters: submitter not supplied",
		},
		{
			name: "Good",
			params: []events.Parameter{
				events.WithLogLevel(zerolog.Disabled),
				events.WithChainTime(chainTime),
				events.WithEventsProviders(map[string]consensusclient.EventsProvider{
					"test": mockClient,
				}),
				events.WithSubmitter(submitter),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := events.New(context.Background(), test.params...)
			if test.err != "" {
				require.EqualError(t, err, test.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
