package workflowapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloud-ru/evolution-devservices-cli/internal/workflowapi"
)

func TestDo_SuccessDecodesJSONAndSendsAPIKey(t *testing.T) {
	var gotAPIKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-KEY")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"run-1","status":"running"}`))
	}))
	defer srv.Close()

	c := workflowapi.New(srv.URL, "test-api-key", "proj-1")

	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	err := c.Do(context.Background(), http.MethodGet, "/run/run-1", nil, nil, &out)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if out.ID != "run-1" || out.Status != "running" {
		t.Errorf("decoded = %+v, want id=run-1 status=running", out)
	}
	if gotAPIKey != "test-api-key" {
		t.Errorf("X-API-KEY header = %q, want %q", gotAPIKey, "test-api-key")
	}
	if gotPath != "/run/run-1" {
		t.Errorf("request path = %q, want %q", gotPath, "/run/run-1")
	}
}

func TestDo_SetTokenChangesSubsequentRequests(t *testing.T) {
	var gotAPIKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-KEY")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := workflowapi.New(srv.URL, "old-key", "proj-1")
	c.SetToken("new-key")

	if err := c.Do(context.Background(), http.MethodGet, "/run/x", nil, nil, nil); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if gotAPIKey != "new-key" {
		t.Errorf("X-API-KEY header = %q, want %q (after SetToken)", gotAPIKey, "new-key")
	}
}

func TestDo_ErrorShape_ErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid pipeline"}`))
	}))
	defer srv.Close()

	c := workflowapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodPost, "/pipeline/x/run", nil, map[string]string{}, nil)

	var apiErr *workflowapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *workflowapi.APIError", err)
	}
	if apiErr.Message != "invalid pipeline" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "invalid pipeline")
	}
}

func TestDo_ErrorShape_ValidationErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":{"branch":["is required"]}}`))
	}))
	defer srv.Close()

	c := workflowapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodPost, "/application", nil, map[string]string{}, nil)

	var apiErr *workflowapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *workflowapi.APIError", err)
	}
	if apiErr.Message != "branch: is required" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "branch: is required")
	}
}

func TestDo_ErrorShape_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := workflowapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodGet, "/run/x", nil, nil, nil)

	var apiErr *workflowapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *workflowapi.APIError", err)
	}
	if got := apiErr.Error(); got != "workflow api error 500: (empty response body)" {
		t.Errorf("Error() = %q, want %q", got, "workflow api error 500: (empty response body)")
	}
}

func TestDo_RequestIDIsCapturedFromHeaderAndIncludedInError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "wf-req-7")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream down"}`))
	}))
	defer srv.Close()

	c := workflowapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodGet, "/run/x", nil, nil, nil)

	var apiErr *workflowapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *workflowapi.APIError", err)
	}
	if apiErr.RequestID != "wf-req-7" {
		t.Errorf("RequestID = %q, want %q", apiErr.RequestID, "wf-req-7")
	}
	want := "workflow api error 502: upstream down (request-id: wf-req-7)"
	if got := apiErr.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestNew_ExposesProjectID(t *testing.T) {
	c := workflowapi.New("https://example.invalid", "key", "proj-42")
	if got := c.ProjectID(); got != "proj-42" {
		t.Errorf("ProjectID() = %q, want %q", got, "proj-42")
	}
}
