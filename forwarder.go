package forwarder

import (
	"context"
	"fmt"
	"github.com/calyptia/cmetrics-go"
	"strconv"
	"strings"
	"time"

	"github.com/calyptia/fluent-bit-cloud-forwarder/cloud"
	fluentbit "github.com/calyptia/go-fluent-bit-metrics"
)

type Forwarder struct {
	Hostname        string
	Store           Store
	Interval        time.Duration
	FluentBitClient FluentBitClient
	CloudClient     CloudClient

	errChan chan error
	nowFunc func() time.Time
}

type Store interface {
	Has(key string) bool
	Write(key string, val []byte) error
	Read(key string) ([]byte, error)
	Erase(key string) error
}

type FluentBitClient interface {
	BuildInfo(ctx context.Context) (fluentbit.BuildInfo, error)
	Metrics(ctx context.Context) (fluentbit.Metrics, error)
}

type CloudClient interface {
	CreateAgent(ctx context.Context, in cloud.CreateAgentInput) (cloud.Agent, error)
	Agent(ctx context.Context, agentID string) (cloud.Agent, error)
	UpsertAgent(ctx context.Context, agentID string, in cloud.UpsertAgentInput) (cloud.Agent, error)
	AddMetrics(ctx context.Context, agentID string, msgPackEncoded []byte) error
}

func (fd *Forwarder) Errs() <-chan error {
	if fd.errChan == nil {
		fd.errChan = make(chan error)
	}

	return fd.errChan
}

func (fd *Forwarder) Forward(ctx context.Context) error {
	buildInfo, err := fd.FluentBitClient.BuildInfo(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch fluent bit build info: %w", err)
	}

	agent, err := fd.CloudClient.CreateAgent(ctx, cloud.CreateAgentInput{
		Name: fd.Hostname,
		Metadata: cloud.AgentMetadata{
			Version: buildInfo.FluentBit.Version,
			Edition: buildInfo.FluentBit.Edition,
			Type:    cloud.AgentMetadataTypeFluentBit,
			Flags:   strings.Join(buildInfo.FluentBit.Flags, ","),
		},
	})
	// TODO: better error handling. Create agent, only if not created already.
	// Otherwise upsert it.
	if err != nil {
		return fmt.Errorf("could not create agent: %w", err)
	}

	ticker := time.NewTicker(fd.Interval)
	if fd.errChan == nil {
		fd.errChan = make(chan error)
	}

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			go func() {
				ctx, cancel := context.WithTimeout(ctx, fd.Interval)
				defer cancel()

				metrics, err := fd.FluentBitClient.Metrics(ctx)
				if err != nil {
					fd.errChan <- fmt.Errorf("could not fetch fluent bit metrics: %w", err)
					return
				}

				msgPackEncoded, err := fd.fluentBitMetricsToCMetrics(&metrics)
				if err != nil {
					fd.errChan <- err
				}

				err = fd.CloudClient.AddMetrics(ctx, strconv.FormatInt(int64(agent.ID), 10), msgPackEncoded)
				if err != nil {
					fd.errChan <- fmt.Errorf("could not push metric to cloud: %w", err)
				}
			}()
		}
	}

	return nil
}

func (fd *Forwarder) fluentBitMetricsToCMetrics(metrics *fluentbit.Metrics) ([]byte, error) {
	if fd.nowFunc == nil {
		fd.nowFunc = time.Now
	}

	ts := fd.nowFunc()

	metricsContext, err := cmetrics.NewContext()
	if err != nil {
		return nil, err
	}

	defer metricsContext.Destroy()

	for metricName, metric := range metrics.Input {
		counter, err := metricsContext.CounterCreate("fluentbit", "input", "records", "records", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.Records), []string{metricName})
		if err != nil {
			return nil, err
		}
		counter, err = metricsContext.CounterCreate("fluentbit", "input", "bytes", "bytes", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.Bytes), []string{metricName})
		if err != nil {
			return nil, err
		}
	}

	for metricName, metric := range metrics.Output {
		counter, err := metricsContext.CounterCreate("fluentbit", "output", "proc_records", "proc_records", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.ProcRecords), []string{metricName})
		if err != nil {
			return nil, err
		}
		counter, err = metricsContext.CounterCreate("fluentbit", "output", "proc_bytes", "proc_bytes", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.ProcBytes), []string{metricName})
		if err != nil {
			return nil, err
		}
		counter, err = metricsContext.CounterCreate("fluentbit", "output", "errors", "errors", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.Errors), []string{metricName})
		if err != nil {
			return nil, err
		}
		counter, err = metricsContext.CounterCreate("fluentbit", "output", "retries", "retries", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.Retries), []string{metricName})
		if err != nil {
			return nil, err
		}
		counter, err = metricsContext.CounterCreate("fluentbit", "output", "retries_failed", "retries_failed", []string{"plugin"})
		if err != nil {
			return nil, err
		}
		err = counter.Set(ts, float64(metric.RetriesFailed), []string{metricName})
		if err != nil {
			return nil, err
		}
	}

	return metricsContext.EncodeMsgPack()
}
