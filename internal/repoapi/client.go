// Package repoapi provides a thin client for the Repo HTTP API.
package repoapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the Repo API and authenticates with X-API-KEY.
type Client struct {
	baseURL   string
	apiKey    string
	projectID string
	http      *http.Client
}

// New returns a configured API client.
func New(baseURL, apiKey, projectID string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		projectID: projectID,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProjectID returns the configured project ID.
func (c *Client) ProjectID() string { return c.projectID }

// APIError is a non-2xx response from the API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
	// RequestID is the server-assigned request id (from a response header
	// such as X-Request-Id), when present. Include it when reporting bugs
	// to the API team -- it lets them find the request in their logs even
	// when the body carries no useful detail (e.g. a bare 500 "{}").
	RequestID string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = strings.TrimSpace(e.Body)
	}
	if msg == "" || msg == "{}" {
		msg = "(empty response body)"
	}
	if e.RequestID != "" {
		return fmt.Sprintf("api error %d: %s (request-id: %s)", e.StatusCode, msg, e.RequestID)
	}
	return fmt.Sprintf("api error %d: %s", e.StatusCode, msg)
}

// Do performs an authenticated HTTP request and decodes JSON into out (if non-nil).
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, u, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	requestID := requestIDFromHeader(resp.Header)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseError(resp.StatusCode, respBody, requestID)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			if requestID != "" {
				return fmt.Errorf("decode response: %w (body: %s, request-id: %s)", err, truncate(string(respBody), 256), requestID)
			}
			return fmt.Errorf("decode response: %w (body: %s)", err, truncate(string(respBody), 256))
		}
	}
	return nil
}

// requestIDFromHeader looks for a server-assigned request id under a few
// common header names. http.Header.Get canonicalizes header names, so this
// also matches any casing the server sends (x-request-id, X-REQUEST-ID, ...).
func requestIDFromHeader(h http.Header) string {
	for _, name := range []string{"X-Request-Id", "X-Trace-Id", "Trace-Id", "Request-Id", "X-Correlation-Id"} {
		if v := h.Get(name); v != "" {
			return v
		}
	}
	return ""
}

func (c *Client) buildURL(path string, query url.Values) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base url %q: %w", c.baseURL, err)
	}
	// baseURL is the API root, e.g. https://host/repo/api/v1
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

func parseError(status int, body []byte, requestID string) error {
	// Try several known error shapes.
	var errResp struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &errResp)

	var msgResp struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &msgResp)

	var validResp struct {
		Errors map[string][]string `json:"errors"`
	}
	_ = json.Unmarshal(body, &validResp)

	msg := msgResp.Message
	if msg == "" {
		msg = errResp.Error
	}
	if msg == "" && len(validResp.Errors) > 0 {
		var parts []string
		for field, errs := range validResp.Errors {
			parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(errs, ", ")))
		}
		msg = strings.Join(parts, "; ")
	}
	return &APIError{StatusCode: status, Message: msg, Body: string(body), RequestID: requestID}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
