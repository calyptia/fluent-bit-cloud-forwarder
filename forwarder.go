package forwarder

import (
	"context"
	"fmt"
	"time"

	"github.com/calyptia/cloud"
	"github.com/calyptia/cmetrics-go"
	fluentbit "github.com/calyptia/go-fluent-bit-metrics"
	"github.com/go-kit/log"
)

type Forwarder struct {
	Hostname        string
	MachineID       string
	RawConfig       string
	Store           Store
	Interval        time.Duration
	FluentBitClient FluentBitClient
	CloudClient     CloudClient
	Logger          log.Logger

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
	SetProjectToken(token string)
	SetAgentToken(token string)
	CreateToken(ctx context.Context) (cloud.ProjectToken, error)
	Tokens(ctx context.Context, last uint64) ([]cloud.ProjectToken, error)
	CreateAgent(ctx context.Context, payload cloud.CreateAgentPayload) (cloud.CreatedAgentPayload, error)
	UpdateAgent(ctx context.Context, agentID string, in cloud.UpdateAgentOpts) error
	AddAgentMetrics(ctx context.Context, agentID string, msgPackEncoded []byte) (cloud.CreatedAgentMetrics, error)
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

	var projectToken cloud.ProjectToken
	projectTokens, err := fd.CloudClient.Tokens(ctx, 1)
	if err != nil {
		return fmt.Errorf("could not fetch cloud tokens: %w", err)
	}

	if len(projectTokens) == 0 {
		projectToken, err = fd.CloudClient.CreateToken(ctx)
		if err != nil {
			return fmt.Errorf("could not create cloud token: %w", err)
		}
	} else {
		projectToken = projectTokens[0]
	}

	_ = fd.Logger.Log("project_token", projectToken.Token)
	fd.CloudClient.SetProjectToken(projectToken.Token)

	createdAgent, err := fd.CloudClient.CreateAgent(ctx, cloud.CreateAgentPayload{
		Name:      fd.Hostname,
		MachineID: fd.MachineID,
		Type:      cloud.AgentTypeFluentBit,
		Version:   buildInfo.FluentBit.Version,
		Edition:   cloud.AgentEdition(buildInfo.FluentBit.Edition),
		Flags:     buildInfo.FluentBit.Flags,
		RawConfig: fd.RawConfig,
	})
	if err != nil {
		return fmt.Errorf("could not create agent: %w", err)
	}

	// TODO: update agent if already exists.

	_ = fd.Logger.Log(
		"agent_id", createdAgent.ID,
		"agent_token", createdAgent.Token,
		"agent_name", createdAgent.Name,
	)
	fd.CloudClient.SetAgentToken(createdAgent.Token)

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
					fd.errChan <- fmt.Errorf("could not transform fluentbit metrics into cmetrics msgpack")
				}

				_, err = fd.CloudClient.AddAgentMetrics(ctx, createdAgent.ID, msgPackEncoded)
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
