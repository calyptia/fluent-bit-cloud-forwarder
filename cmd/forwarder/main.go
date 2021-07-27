package main

import (
	"context"
	"flag"
	"fmt"
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

const dataPath = "data"

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
		cloudURL             = env("CLOUD_URL", "https://cloud-api-dev.calyptia.com/")
		projectToken         = os.Getenv("PROJECT_TOKEN")
		agentURL             = env("AGENT_URL", "http://localhost:2020")
		agentPullInterval, _ = time.ParseDuration(env("AGENT_PULL_INTERVAL", (time.Second * 5).String()))
		agentHostname        = os.Getenv("AGENT_HOSTNAME")
		agentMachineID       = env("AGENT_MACHINE_ID", func() string { s, _ := machineid.ID(); return s }())
		agentConfigFile      = env("AGENT_CONFIG_FILE", "fluent-bit.conf")
	)

	fs := flag.NewFlagSet("forwarder", flag.ExitOnError)
	fs.StringVar(&cloudURL, "cloud-url", cloudURL, "Calyptia Cloud API URL")
	fs.StringVar(&projectToken, "project-token", projectToken, `Project token from Calyptia Cloud fetched from "POST /api/v1/tokens" or from "GET /api/v1/tokens?last=1"`)
	fs.StringVar(&agentURL, "agent-url", agentURL, "Fluent Bit agent URL")
	fs.DurationVar(&agentPullInterval, "agent-pull-interval", agentPullInterval, "Interval to pull Fluent Bit agent and forward metrics to Cloud")
	fs.StringVar(&agentConfigFile, "agent-config-file", agentConfigFile, "Fluentbit agent config file")
	fs.StringVar(&agentHostname, "agent-hostname", agentHostname, "Agent hostname. If empty, a random one will be generated")
	fs.StringVar(&agentMachineID, "agent-machine-id", agentMachineID, "Agent host machine ID. If empty, a random one will be generated")
	fs.Usage = func() {
		fmt.Printf("Forwards metrics from Fluent Bit agent to Calyptia Cloud.\nIt stores some persisted data about Cloud registration at %q directory.\n", dataPath)
		fmt.Println("Flags:")
		fs.PrintDefaults()
	}

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	if agentHostname == "" {
		rng, err := codename.DefaultRNG()
		if err != nil {
			return fmt.Errorf("could not generate hostname random seed: %w", err)
		}

		agentHostname = codename.Generate(rng, 4)
		_ = logger.Log("generated_hostname", agentHostname)
	}

	if agentMachineID == "" {
		v, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("could not generate random machine ID: %w", err)
		}

		agentMachineID = v.String()
		_ = logger.Log("generated_machine_id", agentMachineID)
	}

	var rawConfig string
	if agentConfigFile != "" {
		b, err := os.ReadFile(agentConfigFile)
		if err != nil {
			return fmt.Errorf("could not read file %q: %w", agentConfigFile, err)
		}

		rawConfig = string(b)
	}

	kv := diskv.New(diskv.Options{
		BasePath: dataPath,
	})

	fd := forwarder.Forwarder{
		Hostname:  agentHostname,
		MachineID: agentMachineID,
		RawConfig: rawConfig,
		Store:     kv,
		Interval:  agentPullInterval,
		FluentBitClient: &fluentbit.Client{
			HTTPClient: http.DefaultClient,
			BaseURL:    agentURL,
		},
		CloudClient: &cloud.Client{
			HTTPClient:   http.DefaultClient,
			BaseURL:      cloudURL,
			ProjectToken: projectToken,
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

func env(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return v
}
