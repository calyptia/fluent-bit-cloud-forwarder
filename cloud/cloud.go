package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/calyptia/cloud"
)

type Error struct {
	Msg string `json:"error"`
}

func (e *Error) Error() string {
	return e.Msg
}

type Client struct {
	BaseURL      string
	HTTPClient   *http.Client
	ProjectToken string
	agentToken   string
}

func (c *Client) SetAgentToken(token string) {
	c.agentToken = token
}

func (c *Client) CreateAgent(ctx context.Context, payload cloud.CreateAgentPayload) (cloud.CreatedAgentPayload, error) {
	var out cloud.CreatedAgentPayload

	if c.ProjectToken == "" {
		return out, errors.New("project token not set yet")
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return out, fmt.Errorf("could not json marshal create agent payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/agents", bytes.NewReader(b))
	if err != nil {
		return out, fmt.Errorf("could not create request to create agent: %w", err)
	}

	req.Header.Set("X-Project-Token", c.ProjectToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("could not do request to create agent: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &Error{}
		err = json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return out, fmt.Errorf("could not json decode create agent error response: %w", err)
		}

		return out, e
	}

	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("could not json decode create agent response: %w", err)
	}

	return out, nil
}

func (c *Client) UpdateAgent(ctx context.Context, agentID string, in cloud.UpdateAgentOpts) error {
	if c.agentToken == "" {
		return errors.New("agent token not set yet")
	}

	b, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("could not json marshal update agent options: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.BaseURL+"/api/v1/agents/"+url.PathEscape(agentID), bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("could not create request to update agent: %w", err)
	}

	req.Header.Set("X-Agent-Token", c.agentToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not do request to update agent: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &Error{}
		err = json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return fmt.Errorf("could not json decode update agent error response: %w", err)
		}

		return e
	}

	return nil
}

func (c *Client) AddAgentMetrics(ctx context.Context, agentID string, msgPackEncoded []byte) (cloud.CreatedAgentMetrics, error) {
	var out cloud.CreatedAgentMetrics

	if c.agentToken == "" {
		return out, errors.New("agent token not set yet")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/agents/"+url.PathEscape(agentID)+"/metrics", bytes.NewReader(msgPackEncoded))
	if err != nil {
		return out, fmt.Errorf("could not create request to add agent metrics: %w", err)
	}

	req.Header.Set("X-Agent-Token", c.agentToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("could not do request to add agent metrics: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e := &Error{}
		err = json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return out, fmt.Errorf("could not json decode add agent metrics error response: %w", err)
		}

		return out, e
	}

	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("could not json decode add agent metrics response: %w", err)
	}

	return out, nil
}
