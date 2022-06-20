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

package main

import (
	"context"
	"sync"

	eth2client "github.com/attestantio/go-eth2-client"
	httpclient "github.com/attestantio/go-eth2-client/http"
	"github.com/pkg/errors"
	"github.com/wealdtech/probec/util"
)

var clients map[string]eth2client.Service
var clientsMu sync.RWMutex

// fetchClient fetches a client service, instantiating it if required.
func fetchClient(ctx context.Context, address string) (eth2client.Service, error) {
	clientsMu.RLock()
	if clients == nil {
		clients = make(map[string]eth2client.Service)
	}
	client, exists := clients[address]
	clientsMu.RUnlock()

	if !exists {
		var err error
		client, err = httpclient.New(ctx,
			httpclient.WithLogLevel(util.LogLevel("consensusclient")),
			httpclient.WithTimeout(util.Timeout("consensusclient")),
			httpclient.WithAddress(address))
		if err != nil {
			return nil, errors.Wrap(err, "failed to initiate client")
		}
		clientsMu.Lock()
		clients[address] = client
		clientsMu.Unlock()
	}
	return client, nil
}
