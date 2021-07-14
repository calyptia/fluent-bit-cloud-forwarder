package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	forwarder "github.com/calyptia/fluent-bit-cloud-forwarder"
	"github.com/calyptia/fluent-bit-cloud-forwarder/cloud"
	fluentbit "github.com/calyptia/go-fluent-bit-metrics"
	"github.com/denisbrodbeck/machineid"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/lucasepe/codename"
	"github.com/peterbourgon/diskv"
)

const diskvBasePath = "diskv-data"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	err := run(ctx, logger, os.Args[1:])
	if err != nil {
		_ = logger.Log("err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger log.Logger, args []string) error {
	var (
		agentURL     string
		interval     time.Duration
		cloudURL     string
		accessToken  string
		hostname     = os.Getenv("HOSTNAME")
		machineID, _ = machineid.ID()
		configFile   string
	)
	fs := flag.NewFlagSet("forwarder", flag.ExitOnError)
	fs.StringVar(&agentURL, "agent", "http://localhost:2020", "Fluent Bit agent URL")
	fs.DurationVar(&interval, "interval", time.Second*5, "Interval to pull Fluent Bit agent and forward metrics to Cloud")
	fs.StringVar(&cloudURL, "cloud", "http://localhost:5000", "Calyptia Cloud API URL")
	fs.StringVar(&accessToken, "access-token", "", "Calyptia Cloud access token taken from auth0")
	fs.StringVar(&hostname, "hostname", hostname, "Agent hostname. If empty, a random one will be generated")
	fs.StringVar(&machineID, "machine-id", machineID, "Machine ID. If empty, a random one will be generated")
	fs.StringVar(&configFile, "config", configFile, "Fluentbit config file")
	fs.Usage = func() {
		fmt.Printf("Forwards metrics from Fluent Bit agent to Calyptia Cloud.\nIt stores some persisted data about Cloud registration at %q directory.\n", diskvBasePath)
		fmt.Println("Flags:")
		fs.PrintDefaults()
	}

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	if hostname == "" {
		rng, err := codename.DefaultRNG()
		if err != nil {
			return fmt.Errorf("could not generate hostname random seed: %w", err)
		}

		hostname = codename.Generate(rng, 4)
		_ = logger.Log("generated_hostname", hostname)
	}

	if machineID == "" {
		v, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("could not generate random machine ID: %w", err)
		}

		machineID = v.String()
		_ = logger.Log("generated_machine_id", machineID)
	}

	var rawConfig string
	if configFile != "" {
		f, err := os.Open(configFile)
		if err != nil {
			return fmt.Errorf("could not open %q: %w", configFile, err)
		}

		defer f.Close()

		b, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("could not read config file contents: %w", err)
		}

		rawConfig = string(b)
	}

	kv := diskv.New(diskv.Options{
		BasePath: diskvBasePath,
	})

	fd := forwarder.Forwarder{
		Hostname:  hostname,
		MachineID: machineID,
		RawConfig: rawConfig,
		Store:     kv,
		Interval:  interval,
		FluentBitClient: &fluentbit.Client{
			HTTPClient: http.DefaultClient,
			BaseURL:    agentURL,
		},
		CloudClient: &cloud.Client{
			HTTPClient:  http.DefaultClient,
			BaseURL:     cloudURL,
			AccessToken: accessToken,
		},
		Logger: logger,
	}

	go func() {
		for err := range fd.Errs() {
			_ = logger.Log("err", err)
		}
	}()

	return fd.Forward(ctx)
}
