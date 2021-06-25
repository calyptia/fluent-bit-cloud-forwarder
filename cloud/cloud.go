package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const apiKeyHeader = "X-API-Key"

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
}

type Errors struct {
	StatusCode int      `json:"-"`
	Errs       []string `json:"errors"`
}

func (e *Errors) Error() string {
	if len(e.Errs) == 0 {
		return ""
	}

	return e.Errs[0]
}

type CreateAgentInput struct {
	Name     string        `json:"name"`
	Metadata AgentMetadata `json:"metadata"`
	Config   string        `json:"config"`
}

type UpsertAgentInput struct {
	Status   string        `json:"status"`
	Metadata AgentMetadata `json:"metadata"`
	Config   string        `json:"config"`
}

type AddMetricsInput struct {
	Metrics []AddMetricInput `json:"metrics"`
}

type AddMetricInput struct {
	Type    MetricType    `json:"type"`
	Options MetricOpts    `json:"opts"`
	Labels  []string      `json:"labels"`
	Values  []MetricValue `json:"values"`
}

type MetricType int

const (
	MetricTypeCounter MetricType = iota
	MetricTypeGauge
	MetricTypeHistogram
)

type MetricOpts struct {
	Namespace string
	Subsystem string
	Name      string
	FQName    string
}

type MetricValue struct {
	Timestamp int64   `json:"ts"`
	Value     float64 `json:"value"`
	Labels    []int64 `json:"labels"`
}

func (v MetricValue) TimestampAsTime() time.Time {
	return time.Unix(0, v.Timestamp)
}

type Agent struct {
	ID     uint        `json:"id"`
	Status AgentStatus `json:"status"`
	Name   string      `json:"name"`
	UserID uint        `json:"user_id"`
	Config string      `json:"config"`
	AgentMetadata
}

type AgentMetadata struct {
	Version string            `json:"version"`
	Edition string            `json:"edition"`
	Type    AgentMetadataType `json:"type"`
	Flags   string            `json:"flags"`
}

type AgentStatus string

const (
	AgentStatusNew     AgentStatus = "NEW"
	AgentStatusRunning AgentStatus = "RUNNING"
	AgentStatusDeleted AgentStatus = "DELETED"
	AgentStatusError   AgentStatus = "ERROR"
)

type AgentMetadataType string

const (
	AgentMetadataTypeFluentd   AgentMetadataType = "fluentd"
	AgentMetadataTypeFluentBit AgentMetadataType = "fluentbit"
)

func (c *Client) CreateAgent(ctx context.Context, in CreateAgentInput) (Agent, error) {
	var out Agent

	b, err := json.Marshal(in)
	if err != nil {
		return out, fmt.Errorf("could not json marshal create agent input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/agent", bytes.NewReader(b))
	if err != nil {
		return out, fmt.Errorf("could not create request to create agent: %w", err)
	}

	req.Header.Set(apiKeyHeader, c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("could not do request to create agent: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errs := &Errors{StatusCode: resp.StatusCode}
		err = json.NewDecoder(resp.Body).Decode(&errs)
		if err != nil {
			return out, fmt.Errorf("could not json decode create agent error response: %w", err)
		}

		return out, errs
	}

	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("could not json decode create agent response: %w", err)
	}

	return out, nil
}

func (c *Client) Agent(ctx context.Context, agentID string) (Agent, error) {
	var out Agent

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/agent/"+url.PathEscape(agentID), nil)
	if err != nil {
		return out, fmt.Errorf("could not create request to retrieve agent: %w", err)
	}

	req.Header.Set(apiKeyHeader, c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("could not do request to retrieve agent: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errs := &Errors{StatusCode: resp.StatusCode}
		err = json.NewDecoder(resp.Body).Decode(&errs)
		if err != nil {
			return out, fmt.Errorf("could not json decode agent error response: %w", err)
		}

		return out, errs
	}

	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("could not json decode agent response: %w", err)
	}

	return out, nil
}

func (c *Client) UpsertAgent(ctx context.Context, agentID string, in UpsertAgentInput) (Agent, error) {
	var out Agent

	b, err := json.Marshal(in)
	if err != nil {
		return out, fmt.Errorf("could not json marshal upsert agent input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+"/api/v1/agent/"+url.PathEscape(agentID), bytes.NewReader(b))
	if err != nil {
		return out, fmt.Errorf("could not create request to upsert agent: %w", err)
	}

	req.Header.Set(apiKeyHeader, c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("could not do request to upsert agent: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errs := &Errors{StatusCode: resp.StatusCode}
		err = json.NewDecoder(resp.Body).Decode(&errs)
		if err != nil {
			return out, fmt.Errorf("could not json decode upsert agent error response: %w", err)
		}

		return out, errs
	}

	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("could not json decode upsert agent response: %w", err)
	}

	return out, nil
}

func (c *Client) AddMetrics(ctx context.Context, agentID string, in AddMetricsInput) error {
	b, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("could not json marshal add agent metric input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+"/api/v1/agent/"+url.PathEscape(agentID)+"/metric", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("could not create request to add agent metric: %w", err)
	}

	req.Header.Set(apiKeyHeader, c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not do request to add agent metric: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errs := &Errors{StatusCode: resp.StatusCode}
		err = json.NewDecoder(resp.Body).Decode(&errs)
		if err != nil {
			return fmt.Errorf("could not json decode add agent metric error response: %w", err)
		}

		return errs
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("could not discard add agent metric response: %w", err)
	}

	return nil
}
