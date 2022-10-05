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
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	consensusclient "github.com/attestantio/go-eth2-client"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	zerologger "github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	standardchaintime "github.com/wealdtech/chaind/services/chaintime/standard"
	eventsattestations "github.com/wealdtech/probec/services/attestations/events"
	eventsblocks "github.com/wealdtech/probec/services/blocks/events"
	eventsheads "github.com/wealdtech/probec/services/heads/events"
	"github.com/wealdtech/probec/services/metrics"
	nullmetrics "github.com/wealdtech/probec/services/metrics/null"
	prometheusmetrics "github.com/wealdtech/probec/services/metrics/prometheus"
	immediatesubmitter "github.com/wealdtech/probec/services/submitter/immediate"
	"github.com/wealdtech/probec/util"
)

// ReleaseVersion is the release version for the code.
var ReleaseVersion = "0.3.0"

func main() {
	os.Exit(main2())
}

func main2() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := fetchConfig(); err != nil {
		zerologger.Error().Err(err).Msg("failed to fetch configuration")
		return 1
	}

	if err := initLogging(); err != nil {
		log.Error().Err(err).Msg("Failed to initialise logging")
		return 1
	}

	// runCommands will not return if a command is run.
	runCommands(ctx)

	logModules()
	log.Info().Str("version", ReleaseVersion).Msg("Starting probec")

	runtime.GOMAXPROCS(runtime.NumCPU() * 8)

	log.Trace().Msg("Starting metrics service")
	monitor, err := startMonitor(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start metrics service")
		return 1
	}
	if err := registerMetrics(ctx, monitor); err != nil {
		log.Error().Err(err).Msg("Failed to register metrics")
		return 1
	}
	setRelease(ctx, ReleaseVersion)
	setReady(ctx, false)

	if err := startServices(ctx, monitor); err != nil {
		log.Error().Err(err).Msg("Failed to initialise services")
		return 1
	}
	setReady(ctx, true)

	log.Info().Msg("All services operational")

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh
		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}

	log.Info().Msg("Stopping probec")
	return 0
}

// fetchConfig fetches configuration from various sources.
func fetchConfig() error {
	pflag.String("base-dir", "", "base directory for configuration files")
	pflag.Bool("version", false, "show version and exit")
	pflag.String("log-level", "info", "minimum level of messsages to log")
	pflag.String("log-file", "", "redirect log output to a file")
	pflag.Bool("blocks.enable", true, "enable logging of block delays")
	pflag.Bool("heads.enable", true, "enable logging of head delays")
	pflag.Bool("attestations.enable", true, "enable logging of attestations and their delays")
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return errors.Wrap(err, "failed to bind pflags to viper")
	}

	if viper.GetString("base-dir") != "" {
		// User-defined base directory.
		viper.AddConfigPath(util.ResolvePath(""))
		viper.SetConfigName("execd")
	} else {
		// Home directory.
		home, err := homedir.Dir()
		if err != nil {
			return errors.Wrap(err, "failed to obtain home directory")
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".probec")
	}

	// Environment settings.
	viper.SetEnvPrefix("PROBEC")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// Defaults.
	viper.SetDefault("consensusclient.timeout", 2*time.Minute)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return errors.Wrap(err, "failed to read configuration file")
		}
	}

	return nil
}

func startMonitor(ctx context.Context) (metrics.Service, error) {
	var monitor metrics.Service
	if viper.Get("metrics.prometheus.listen-address") != nil {
		var err error
		monitor, err = prometheusmetrics.New(ctx,
			prometheusmetrics.WithLogLevel(util.LogLevel("metrics.prometheus")),
			prometheusmetrics.WithAddress(viper.GetString("metrics.prometheus.listen-address")),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to start prometheus metrics service")
		}
		log.Info().Str("listen_address", viper.GetString("metrics.prometheus.listen-address")).Msg("Started prometheus metrics service")
	} else {
		log.Debug().Msg("No metrics service supplied; monitor not starting")
		monitor = &nullmetrics.Service{}
	}
	return monitor, nil
}

func startServices(ctx context.Context, monitor metrics.Service) error {
	submitter, err := immediatesubmitter.New(ctx,
		immediatesubmitter.WithLogLevel(util.LogLevel("submitter.immediate")),
		immediatesubmitter.WithMonitor(monitor),
		immediatesubmitter.WithBaseUrl(viper.GetString("submitter.base-url")),
	)
	if err != nil {
		return errors.Wrap(err, "failed to start submitter")
	}

	// Obtain providers.
	addresses := viper.GetStringSlice("consensusclient.addresses")
	if len(addresses) == 0 {
		return errors.New("no consensus client addresses provided")
	}
	providers := make(map[string]consensusclient.EventsProvider)
	var firstClient consensusclient.Service
	for _, address := range addresses {
		client, err := fetchClient(ctx, address)
		if err != nil {
			return errors.Wrap(err, "failed to fetch client")
		}
		eventsProvider, isProvider := client.(consensusclient.EventsProvider)
		if !isProvider {
			return fmt.Errorf("%s does not provide events", address)
		}
		nodeVersion, err := client.(consensusclient.NodeVersionProvider).NodeVersion(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to fetch node version")
		}
		providers[nodeVersion] = eventsProvider
		if firstClient == nil {
			firstClient = client
		}
	}

	chainTime, err := standardchaintime.New(ctx,
		standardchaintime.WithGenesisTimeProvider(firstClient.(consensusclient.GenesisTimeProvider)),
		standardchaintime.WithSpecProvider(firstClient.(consensusclient.SpecProvider)),
		standardchaintime.WithForkScheduleProvider(firstClient.(consensusclient.ForkScheduleProvider)),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create chain time service")
	}

	if viper.GetBool("blocks.enable") {
		log.Trace().Msg("Starting blocks service")
		if _, err := eventsblocks.New(ctx,
			eventsblocks.WithLogLevel(util.LogLevel("blocks.events")),
			eventsblocks.WithMonitor(monitor),
			eventsblocks.WithChainTime(chainTime),
			eventsblocks.WithEventsProviders(providers),
			eventsblocks.WithSubmitter(submitter),
		); err != nil {
			return err
		}
	}

	if viper.GetBool("heads.enable") {
		log.Trace().Msg("Starting heads service")
		if _, err := eventsheads.New(ctx,
			eventsheads.WithLogLevel(util.LogLevel("heads.events")),
			eventsheads.WithMonitor(monitor),
			eventsheads.WithChainTime(chainTime),
			eventsheads.WithEventsProviders(providers),
			eventsheads.WithSubmitter(submitter),
		); err != nil {
			return err
		}
	}

	if viper.GetBool("attestations.enable") {
		log.Trace().Msg("Starting attestations service")
		if _, err := eventsattestations.New(ctx,
			eventsattestations.WithLogLevel(util.LogLevel("attestations.events")),
			eventsattestations.WithMonitor(monitor),
			eventsattestations.WithChainTime(chainTime),
			eventsattestations.WithEventsProviders(providers),
			eventsattestations.WithSubmitter(submitter),
		); err != nil {
			return err
		}
	}

	return nil
}

func logModules() {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		log.Trace().Str("path", buildInfo.Path).Msg("Main package")
		for _, dep := range buildInfo.Deps {
			log := log.Trace()
			if dep.Replace == nil {
				log = log.Str("path", dep.Path).Str("version", dep.Version)
			} else {
				log = log.Str("path", dep.Replace.Path).Str("version", dep.Replace.Version)
			}
			log.Msg("Dependency")
		}
	}
}

func runCommands(ctx context.Context) {
	if viper.GetBool("version") {
		fmt.Printf("%s\n", ReleaseVersion)
		os.Exit(0)
	}
}
