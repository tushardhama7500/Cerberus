package authz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an OPA authorization client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new OPA client with the given base URL.
// baseURL should be like "http://localhost:8181"
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		timeout: 5 * time.Second,
	}
}

// OPAInput represents the input data sent to OPA for policy evaluation.
type OPAInput struct {
	Action   string                 `json:"action"`
	Resource string                 `json:"resource"`
	User     map[string]interface{} `json:"user"`
	Data     map[string]interface{} `json:"data,omitempty"` // Additional context
}

// OPARequest represents the complete request payload sent to OPA.
type OPARequest struct {
	Input OPAInput `json:"input"`
}

// OPAResponse represents the response from OPA.
type OPAResponse struct {
	Result OPADecision `json:"result"`
}

// OPADecision contains the authorization decision.
type OPADecision struct {
	Allow   bool                   `json:"allow"`
	Reason  string                 `json:"reason,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Allow checks if an action is allowed based on OPA policy evaluation.
// It sends the request to OPA and returns the authorization decision.
//
// Example usage:
//
//	allowed, err := opaClient.Allow(ctx, map[string]any{
//		"action":   "approve_request",
//		"resource": "engineering-system",
//		"user": map[string]any{
//			"email": "user@example.com",
//			"role":  "ENGINEERING",
//		},
//	})
func (c *Client) Allow(
	ctx context.Context,
	input map[string]any,
) (bool, error) {
	// Extract fields from input map
	action, _ := input["action"].(string)
	resource, _ := input["resource"].(string)
	user, _ := input["user"].(map[string]interface{})
	data, _ := input["data"].(map[string]interface{})

	opaInput := OPAInput{
		Action:   action,
		Resource: resource,
		User:     user,
		Data:     data,
	}

	req := OPARequest{
		Input: opaInput,
	}

	decision, err := c.Evaluate(ctx, "authz/allow", req)
	if err != nil {
		return false, err
	}

	return decision.Allow, nil
}

// Evaluate sends a request to OPA and returns the decision.
// path is the OPA policy path, e.g., "authz/allow"
func (c *Client) Evaluate(
	ctx context.Context,
	path string,
	req OPARequest,
) (*OPADecision, error) {
	// Create context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OPA request: %w", err)
	}

	// Build OPA URL
	url := fmt.Sprintf("%s/v1/data/%s", c.baseURL, path)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(evalCtx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPA response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var opaResp OPAResponse
	if err := json.Unmarshal(respBody, &opaResp); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	return &opaResp.Result, nil
}

// Health checks if OPA is available and responding.
func (c *Client) Health(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(healthCtx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OPA health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OPA health check returned status %d", resp.StatusCode)
	}

	return nil
}
