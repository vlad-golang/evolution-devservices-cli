package workflowapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SortOrder is the sort parameter accepted by the application/deployment
// list endpoints.
type SortOrder string

const (
	SortCreatedAtAsc  SortOrder = "created_at_asc"
	SortCreatedAtDesc SortOrder = "created_at_desc"
)

// Application is a Workflow Studio service: a repository + branch wired to a
// deploy pipeline. Creating a deployment for it runs that pipeline and
// publishes the result.
type Application struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Branch        string `json:"branch"`
	ProjectID     string `json:"project_id"`
	RepositoryID  string `json:"repository_id"`
	RepositoryURL string `json:"repository_url"`
	PipelineID    string `json:"pipeline_id"`
	TemplateID    string `json:"template_id"`
	RunID         string `json:"run_id"`
	Run           *Run   `json:"run,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ApplicationListResponse is the response payload of GET /application/list.
type ApplicationListResponse struct {
	Applications []Application `json:"applications"`
	Total        int           `json:"total"`
}

// CreateApplicationRequest is the body for POST /application. Either
// RepositoryID (a repo already known to the Repo service, see
// internal/repoapi) or RepositoryURL (an external git URL) must be set.
type CreateApplicationRequest struct {
	Name          string `json:"name,omitempty"`
	Branch        string `json:"branch"`
	RepositoryID  string `json:"repository_id,omitempty"`
	RepositoryURL string `json:"repository_url,omitempty"`
}

// UpdateApplicationRequest is the body for PATCH /application/{id}.
type UpdateApplicationRequest struct {
	Name   string `json:"name,omitempty"`
	Branch string `json:"branch"`
}

// Deployment represents a single publish/run of an application's pipeline.
// URL is the published site/service URL once the deployment succeeds.
type Deployment struct {
	ID            string       `json:"id"`
	ApplicationID string       `json:"application_id"`
	Application   *Application `json:"application,omitempty"`
	RunID         string       `json:"run_id"`
	Run           *Run         `json:"run,omitempty"`
	ShortName     string       `json:"short_name"`
	URL           string       `json:"url"`
	CreatedAt     string       `json:"created_at"`
}

// CreateDeploymentResponse is the response payload of POST /application/{id}/deployment.
type CreateDeploymentResponse struct {
	Deployment Deployment `json:"deployment"`
}

// DeploymentListResponse is the response payload of GET /application/{id}/deployment/list.
type DeploymentListResponse struct {
	Deployments []Deployment `json:"deployments"`
	Total       int          `json:"total"`
}

// ListApplicationsOptions configures ListApplications.
type ListApplicationsOptions struct {
	Search string
	Sort   SortOrder
	Limit  int
	Offset int
}

// ListDeploymentsOptions configures ListDeployments.
type ListDeploymentsOptions struct {
	Search string
	Sort   SortOrder
	Limit  int
	Offset int
}

// ListApplications lists applications in the configured project.
func (c *Client) ListApplications(ctx context.Context, opts ListApplicationsOptions) (*ApplicationListResponse, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	q := url.Values{}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.Sort != "" {
		q.Set("sort", string(opts.Sort))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	var out ApplicationListResponse
	if err := c.Do(ctx, "GET", fmt.Sprintf("/project/%s/application/list", c.projectID), q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetApplication fetches a single application by id, including its latest run.
func (c *Client) GetApplication(ctx context.Context, id string) (*Application, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	var out Application
	if err := c.Do(ctx, "GET", fmt.Sprintf("/project/%s/application/%s", c.projectID, id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateApplication creates a new application from a repository + branch.
func (c *Client) CreateApplication(ctx context.Context, req CreateApplicationRequest) (*Application, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	var out Application
	if err := c.Do(ctx, "POST", fmt.Sprintf("/project/%s/application", c.projectID), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateApplication updates an application's name/branch.
func (c *Client) UpdateApplication(ctx context.Context, id string, req UpdateApplicationRequest) error {
	if c.projectID == "" {
		return fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	return c.Do(ctx, "PATCH", fmt.Sprintf("/project/%s/application/%s", c.projectID, id), nil, req, nil)
}

// DeleteApplication removes an application and its deployments.
func (c *Client) DeleteApplication(ctx context.Context, id string) error {
	if c.projectID == "" {
		return fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	return c.Do(ctx, "DELETE", fmt.Sprintf("/project/%s/application/%s", c.projectID, id), nil, nil, nil)
}

// CreateDeployment runs the application's pipeline and publishes the result.
// This is the "deploy" / "publish" action for a Workflow Studio application.
func (c *Client) CreateDeployment(ctx context.Context, applicationID string) (*Deployment, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	var out CreateDeploymentResponse
	err := c.Do(ctx, "POST",
		fmt.Sprintf("/project/%s/application/%s/deployment", c.projectID, applicationID),
		nil, struct{}{}, &out)
	if err != nil {
		return nil, err
	}
	return &out.Deployment, nil
}

// ListDeployments lists deployments for an application, most recent first
// when Sort is SortCreatedAtDesc.
func (c *Client) ListDeployments(ctx context.Context, applicationID string, opts ListDeploymentsOptions) (*DeploymentListResponse, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	q := url.Values{}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.Sort != "" {
		q.Set("sort", string(opts.Sort))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}

	var out DeploymentListResponse
	err := c.Do(ctx, "GET",
		fmt.Sprintf("/project/%s/application/%s/deployment/list", c.projectID, applicationID),
		q, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
