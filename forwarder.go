package forwarder

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	cmetrics "github.com/calyptia/cmetrics-go"
	"github.com/calyptia/fluent-bit-cloud-forwarder/cloud"
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
	SetAgentToken(token string)
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

type StorePayload struct {
	AgentID    string
	AgentToken string
	AgentName  string
}

func (fd *Forwarder) Forward(ctx context.Context) error {
	buildInfo, err := fd.FluentBitClient.BuildInfo(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch fluent bit build info: %w", err)
	}

	var payload StorePayload
	if fd.Store.Has(fd.MachineID) {
		b, err := fd.Store.Read(fd.MachineID)
		if err != nil {
			return fmt.Errorf("could not read from store: %w", err)
		}

		err = gob.NewDecoder(bytes.NewReader(b)).Decode(&payload)
		if err != nil {
			return fmt.Errorf("could not decode store payload: %w", err)
		}

		edition := cloud.AgentEdition(buildInfo.FluentBit.Edition)
		err = fd.CloudClient.UpdateAgent(ctx, payload.AgentID, cloud.UpdateAgentOpts{
			Name:      &fd.Hostname,
			Version:   &buildInfo.FluentBit.Version,
			Edition:   &edition,
			Flags:     &buildInfo.FluentBit.Flags,
			RawConfig: &fd.RawConfig,
		})
		if err != nil {
			return fmt.Errorf("could not update agent: %w", err)
		}
	} else {
		createdAgent, err := fd.CloudClient.CreateAgent(ctx, cloud.CreateAgentPayload{
			Name:      fd.Hostname,
			MachineID: fd.MachineID,
			Type:      cloud.AgentTypeFluentBit,
			Version:   buildInfo.FluentBit.Version,
			Edition:   cloud.AgentEdition(strings.ToLower(buildInfo.FluentBit.Edition)),
			Flags:     buildInfo.FluentBit.Flags,
			RawConfig: fd.RawConfig,
		})
		if err != nil {
			return fmt.Errorf("could not create agent: %w", err)
		}

		payload.AgentID = createdAgent.ID
		payload.AgentToken = createdAgent.Token
		payload.AgentName = createdAgent.Name

		buff := &bytes.Buffer{}
		err = gob.NewEncoder(buff).Encode(payload)
		if err != nil {
			return fmt.Errorf("could not encode store payload: %w", err)
		}

		err = fd.Store.Write(fd.MachineID, buff.Bytes())
		if err != nil {
			return fmt.Errorf("could not write to store: %w", err)
		}
	}

	_ = fd.Logger.Log(
		"agent_id", payload.AgentID,
		"agent_token", payload.AgentToken,
		"agent_name", payload.AgentName,
	)
	fd.CloudClient.SetAgentToken(payload.AgentToken)

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

				_, err = fd.CloudClient.AddAgentMetrics(ctx, payload.AgentID, msgPackEncoded)
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
