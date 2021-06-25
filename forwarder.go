package forwarder

import (
	"context"
	"fmt"
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
	AddMetrics(ctx context.Context, agentID string, in cloud.AddMetricsInput) error
}

func (fd *Forwarder) Errs() <-chan error {
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

				err = fd.CloudClient.AddMetrics(ctx,
					strconv.FormatInt(int64(agent.ID), 10),
					fd.fluentBitMetricsToCloudMetrics(metrics),
				)
				if err != nil {
					fd.errChan <- fmt.Errorf("could not push metric to cloud: %w", err)
				}
			}()
		}
	}

	return nil
}

func (fd *Forwarder) fluentBitMetricsToCloudMetrics(metrics fluentbit.Metrics) cloud.AddMetricsInput {
	var out cloud.AddMetricsInput

	if fd.nowFunc == nil {
		fd.nowFunc = time.Now
	}
	ts := fd.nowFunc().UnixNano()

	for metricName, metric := range metrics.Input {
		out.Metrics = append(out.Metrics, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "input",
				Subsystem: metricName,
				Name:      "records",
				FQName:    strings.Join([]string{"input", metricName, "records"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.Records),
			}},
		}, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "input",
				Subsystem: metricName,
				Name:      "bytes",
				FQName:    strings.Join([]string{"input", metricName, "bytes"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.Bytes),
			}},
		})
	}

	for metricName, metric := range metrics.Output {
		out.Metrics = append(out.Metrics, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "output",
				Subsystem: metricName,
				Name:      "proc_records",
				FQName:    strings.Join([]string{"output", metricName, "proc_records"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.ProcRecords),
			}},
		}, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "output",
				Subsystem: metricName,
				Name:      "proc_bytes",
				FQName:    strings.Join([]string{"output", metricName, "proc_bytes"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.ProcBytes),
			}},
		}, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "output",
				Subsystem: metricName,
				Name:      "errors",
				FQName:    strings.Join([]string{"output", metricName, "errors"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.Errors),
			}},
		}, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "output",
				Subsystem: metricName,
				Name:      "retries",
				FQName:    strings.Join([]string{"output", metricName, "retries"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.Retries),
			}},
		}, cloud.AddMetricInput{
			Type: cloud.MetricTypeCounter,
			Options: cloud.MetricOpts{
				Namespace: "output",
				Subsystem: metricName,
				Name:      "retries_failed",
				FQName:    strings.Join([]string{"output", metricName, "retries_failed"}, "."),
			},
			Values: []cloud.MetricValue{{
				Timestamp: ts,
				Value:     float64(metric.RetriesFailed),
			}},
		})
	}

	return out
}
