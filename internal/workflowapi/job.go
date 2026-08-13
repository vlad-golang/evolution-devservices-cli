package workflowapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// JobStatus is the lifecycle state of a job.
type JobStatus string

const (
	JobStatusPending  JobStatus = "pending"
	JobStatusRunning  JobStatus = "running"
	JobStatusDone     JobStatus = "done"
	JobStatusCanceled JobStatus = "canceled"
	JobStatusFailed   JobStatus = "failed"
)

// Job is a single unit of work within a pipeline run stage.
type Job struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	RunID              string    `json:"run_id"`
	StageID            string    `json:"stage_id"`
	Status             JobStatus `json:"status"`
	Type               string    `json:"type"`
	SequenceNumber     int       `json:"sequence_number"`
	DataPlaneExecName  string    `json:"data_plane_exec_name"`
	DataPlaneJobName   string    `json:"data_plane_job_name"`
	TemporalWorkflowID string    `json:"temporal_workflow_id"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// ListJobsOptions configures ListJobs.
type ListJobsOptions struct {
	RunID  string
	Limit  int
	Offset int
}

// GetJob fetches a single job by id.
func (c *Client) GetJob(ctx context.Context, id string) (*Job, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	var out Job
	if err := c.Do(ctx, "GET", fmt.Sprintf("/project/%s/job/%s", c.projectID, id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListJobs lists jobs belonging to a run.
func (c *Client) ListJobs(ctx context.Context, opts ListJobsOptions) ([]Job, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	if opts.RunID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	q := url.Values{}
	q.Set("run_id", opts.RunID)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	var out []Job
	if err := c.Do(ctx, "GET", fmt.Sprintf("/project/%s/job/list", c.projectID), q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RetryJob re-runs a failed or canceled job.
func (c *Client) RetryJob(ctx context.Context, id string) error {
	if c.projectID == "" {
		return fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	return c.Do(ctx, "POST", fmt.Sprintf("/project/%s/job/%s/retry", c.projectID, id), nil, nil, nil)
}

// StopJob cancels a running job.
func (c *Client) StopJob(ctx context.Context, id string) error {
	if c.projectID == "" {
		return fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	return c.Do(ctx, "POST", fmt.Sprintf("/project/%s/job/%s/stop", c.projectID, id), nil, nil, nil)
}
