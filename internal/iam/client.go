// Package iam exchanges a Workflow Studio key id + secret for a short-lived
// Bearer access token via the cloud.ru IAM service. Workflow Studio does not
// accept the key id/secret directly -- every request needs the exchanged
// access_token in its Authorization header.
package iam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultURL is the cloud.ru IAM token endpoint.
const DefaultURL = "https://iam.api.cloud.ru/api/v1/auth/token"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// TokenResponse is the IAM service's token exchange response.
type TokenResponse struct {
	AccessToken      string   `json:"access_token"`
	IDToken          string   `json:"id_token"`
	RefreshToken     string   `json:"refresh_token"`
	ExpiresIn        int      `json:"expires_in"`
	RefreshExpiresIn int      `json:"refresh_expires_in"`
	Scopes           []string `json:"scopes"`
	TokenType        string   `json:"token_type"`
	NotBefore        int64    `json:"not_before"`
}

// FetchToken exchanges keyID/secret for a Bearer access token. iamURL
// defaults to DefaultURL when empty.
func FetchToken(ctx context.Context, iamURL, keyID, secret string) (*TokenResponse, error) {
	if iamURL == "" {
		iamURL = DefaultURL
	}
	if keyID == "" || secret == "" {
		return nil, fmt.Errorf("workflow key id and secret are both required")
	}

	reqBody, err := json.Marshal(struct {
		KeyID  string `json:"keyId"`
		Secret string `json:"secret"`
	}{KeyID: keyID, Secret: secret})
	if err != nil {
		return nil, fmt.Errorf("marshal iam request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, iamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build iam request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http POST %s: %w", iamURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read iam response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("iam auth failed (%d): %s", resp.StatusCode, truncate(string(body), 256))
	}

	var out TokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode iam response: %w (body: %s)", err, truncate(string(body), 256))
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("iam response did not include an access_token")
	}
	return &out, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
