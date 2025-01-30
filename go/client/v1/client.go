package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	v1 "github.com/substratusai/sandboxai/api/v1"
)

var ErrSandboxNotFound = fmt.Errorf("sandbox not found")

// Client represents a client for interacting with the SandboxAI API.
// See the OpenAPI spec for API details.
type Client struct {
	// BaseURL to send requests to, for example "http://localhost:5000/v1".
	BaseURL string
	httpc   *http.Client
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpc = httpClient
	}
}

func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		BaseURL: baseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) CreateSandbox(ctx context.Context, space string, request *v1.CreateSandboxRequest) (*v1.Sandbox, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/spaces/%s/sandboxes", c.BaseURL, space)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := validateResponse(resp, http.StatusCreated); err != nil {
		return nil, err
	}

	var response v1.Sandbox
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetSandbox(ctx context.Context, space, name string) (*v1.Sandbox, error) {
	url := fmt.Sprintf("%s/spaces/%s/sandboxes/%s", c.BaseURL, space, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrSandboxNotFound
	}
	if err := validateResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var response v1.Sandbox
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) DeleteSandbox(ctx context.Context, space, name string) error {
	url := fmt.Sprintf("%s/spaces/%s/sandboxes/%s", c.BaseURL, space, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := validateResponse(resp, http.StatusNoContent); err != nil {
		return err
	}

	return nil
}

func (c *Client) RunIPythonCell(ctx context.Context, space, name string, request *v1.RunIPythonCellRequest) (*v1.RunIPythonCellResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/spaces/%s/sandboxes/%s/tools:run_ipython_cell", c.BaseURL, space, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := validateResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var response v1.RunIPythonCellResult
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) RunShellCommand(ctx context.Context, space, name string, request *v1.RunShellCommandRequest) (*v1.RunShellCommandResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/spaces/%s/sandboxes/%s/tools:run_shell_command", c.BaseURL, space, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := validateResponse(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var response v1.RunShellCommandResult
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

func validateResponse(resp *http.Response, expectedStatus int) error {
	if resp.StatusCode != expectedStatus {
		plainBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected status %d, got %d: %s", expectedStatus, resp.StatusCode, string(plainBody))
	}
	return nil
}
