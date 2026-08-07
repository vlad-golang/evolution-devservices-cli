package workflowapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// RunStatus is the lifecycle state of a pipeline run.
type RunStatus string

const (
	RunStatusPending  RunStatus = "pending"
	RunStatusRunning  RunStatus = "running"
	RunStatusDone     RunStatus = "done"
	RunStatusCanceled RunStatus = "canceled"
	RunStatusFailed   RunStatus = "failed"
)

// Stage is a group of jobs executed together within a run.
type Stage struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	RunID     string    `json:"run_id"`
	Status    JobStatus `json:"status"`
	Jobs      []Job     `json:"jobs"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// Run is a single execution of a pipeline.
type Run struct {
	ID            string    `json:"id"`
	Branch        string    `json:"branch"`
	PipelineID    string    `json:"pipeline_id"`
	Status        RunStatus `json:"status"`
	StatusMessage string    `json:"status_message"`
	Stages        []Stage   `json:"stages"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

// RunListResponse is the response payload of GET /run/list.
type RunListResponse struct {
	Runs  []Run `json:"runs"`
	Total int   `json:"total"`
}

// ListRunsOptions configures ListRuns.
type ListRunsOptions struct {
	PipelineID string
	Type       string // "cicd" or "workflow"
	Sort       string
	WithConfig bool
	Limit      int
	Offset     int
}

// GetRun fetches a single run by id, including its stages and jobs.
func (c *Client) GetRun(ctx context.Context, id string) (*Run, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	var out Run
	if err := c.Do(ctx, "GET", fmt.Sprintf("/project/%s/run/%s", c.projectID, id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListRuns lists runs in the configured project.
func (c *Client) ListRuns(ctx context.Context, opts ListRunsOptions) (*RunListResponse, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	q := url.Values{}
	if opts.PipelineID != "" {
		q.Set("pipeline_id", opts.PipelineID)
	}
	if opts.Type != "" {
		q.Set("type", opts.Type)
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	if opts.WithConfig {
		q.Set("with_config", "true")
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	var out RunListResponse
	if err := c.Do(ctx, "GET", fmt.Sprintf("/project/%s/run/list", c.projectID), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StopRun cancels a running run.
func (c *Client) StopRun(ctx context.Context, id string) error {
	if c.projectID == "" {
		return fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	return c.Do(ctx, "POST", fmt.Sprintf("/project/%s/run/%s/stop", c.projectID, id), nil, nil, nil)
}
