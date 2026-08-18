package repoapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloud-ru/evolution-devservices-cli/internal/repoapi"
)

func TestDo_SuccessDecodesJSONAndSendsAPIKey(t *testing.T) {
	var gotAPIKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-KEY")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"abc123","name":"demo"}`))
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "test-api-key", "proj-1")

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := c.Do(context.Background(), http.MethodGet, "/repository/abc123", nil, nil, &out)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if out.ID != "abc123" || out.Name != "demo" {
		t.Errorf("decoded = %+v, want id=abc123 name=demo", out)
	}
	if gotAPIKey != "test-api-key" {
		t.Errorf("X-API-KEY header = %q, want %q", gotAPIKey, "test-api-key")
	}
	if gotPath != "/repository/abc123" {
		t.Errorf("request path = %q, want %q", gotPath, "/repository/abc123")
	}
}

func TestDo_ErrorShape_ErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid name"}`))
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodPost, "/repository", nil, map[string]string{"name": ""}, nil)

	var apiErr *repoapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *repoapi.APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if apiErr.Message != "invalid name" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "invalid name")
	}
}

func TestDo_ErrorShape_MessageField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"something broke"}`))
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodGet, "/repository/x", nil, nil, nil)

	var apiErr *repoapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *repoapi.APIError", err)
	}
	if apiErr.Message != "something broke" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "something broke")
	}
}

func TestDo_ErrorShape_ValidationErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":{"name":["is required"]}}`))
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodPost, "/repository", nil, map[string]string{}, nil)

	var apiErr *repoapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *repoapi.APIError", err)
	}
	if apiErr.Message != "name: is required" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "name: is required")
	}
}

func TestDo_ErrorShape_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodGet, "/repository/x", nil, nil, nil)

	var apiErr *repoapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *repoapi.APIError", err)
	}
	if got := apiErr.Error(); got != "api error 500: (empty response body)" {
		t.Errorf("Error() = %q, want %q", got, "api error 500: (empty response body)")
	}
}

func TestDo_RequestIDIsCapturedFromHeaderAndIncludedInError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "req-42")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream down"}`))
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "key", "proj-1")
	err := c.Do(context.Background(), http.MethodGet, "/repository/x", nil, nil, nil)

	var apiErr *repoapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *repoapi.APIError", err)
	}
	if apiErr.RequestID != "req-42" {
		t.Errorf("RequestID = %q, want %q", apiErr.RequestID, "req-42")
	}
	want := "api error 502: upstream down (request-id: req-42)"
	if got := apiErr.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestDo_2xxWithNoOutTargetDoesNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := repoapi.New(srv.URL, "key", "proj-1")
	if err := c.Do(context.Background(), http.MethodDelete, "/repository/x", nil, nil, nil); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestNew_ExposesProjectID(t *testing.T) {
	c := repoapi.New("https://example.invalid", "key", "proj-42")
	if got := c.ProjectID(); got != "proj-42" {
		t.Errorf("ProjectID() = %q, want %q", got, "proj-42")
	}
}
