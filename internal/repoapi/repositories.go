package repoapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// Repository is the lightweight repository listing representation.
type Repository struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Visibility  string `json:"visibility_level"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	CreatorID   string `json:"creator_id"`
	URI         string `json:"uri"`
	ClonePath   string `json:"clone_path"`
	MLHubType   string `json:"mlhub_type"`
}

// RepositoryInfo is the full repository representation (GET by id).
type RepositoryInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	DefaultBranch string `json:"default_branch"`
	Size          int64  `json:"size"`
	BranchesCount int64  `json:"branches_count"`
	CommitsCount  int64  `json:"commits_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Clone         Clone  `json:"clone"`
}

// Clone contains clone URLs (HTTPS and SSH) for a repository.
type Clone struct {
	HTTPS    string `json:"https"`
	SSH      string `json:"ssh"`
	RepoPath string `json:"repo_path"`
}

// ListRepositoriesResponse is the response payload of GET /repositories.
type ListRepositoriesResponse struct {
	Limit        int          `json:"limit"`
	Offset       int          `json:"offset"`
	Total        int          `json:"total"`
	Repositories []Repository `json:"repositories"`
}

// CreateRepositoryRequest is the body for POST /repository.
type CreateRepositoryRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	ProjectID       string `json:"project_id"`
	UserID          string `json:"user_id,omitempty"`
	Type            string `json:"type"`
	VisibilityLevel string `json:"visibility_level,omitempty"`
}

// ListRepositories returns all git repositories of the configured project.
// Search is required by the API; when empty it is replaced with a wildcard
// and all results are returned (paginated).
func (c *Client) ListRepositories(ctx context.Context, opts ListOptions) (*ListRepositoriesResponse, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	q := url.Values{}
	// `search` is documented as required, but the API in practice returns
	// all repositories when it is omitted entirely. Sending an empty
	// value yields an empty list, so we only set it when the caller
	// actually provided something.
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	q.Set("sort", string(opts.Sort))
	if opts.Type != "" {
		q.Set("type", string(opts.Type))
	}
	q.Set("limit", strconv.Itoa(opts.Limit))
	q.Set("offset", strconv.Itoa(opts.Offset))

	var out ListRepositoriesResponse
	err := c.Do(ctx, "GET",
		fmt.Sprintf("/project/%s/repositories", c.projectID),
		q, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRepository fetches a single repository by id.
func (c *Client) GetRepository(ctx context.Context, id string) (*RepositoryInfo, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	var out RepositoryInfo
	err := c.Do(ctx, "GET",
		fmt.Sprintf("/project/%s/repository/%s", c.projectID, id),
		nil, nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateRepository creates a new repository and returns it.
func (c *Client) CreateRepository(ctx context.Context, req CreateRepositoryRequest) (*Repository, error) {
	if c.projectID == "" {
		return nil, fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	req.ProjectID = c.projectID
	var out Repository
	err := c.Do(ctx, "POST",
		fmt.Sprintf("/project/%s/repository", c.projectID),
		nil, req, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteRepository removes a repository. Returns nil on 204.
func (c *Client) DeleteRepository(ctx context.Context, id string) error {
	if c.projectID == "" {
		return fmt.Errorf("project id is not configured; use --project or EDS_PROJECT_ID")
	}
	return c.Do(ctx, "DELETE",
		fmt.Sprintf("/project/%s/repository/%s", c.projectID, id),
		nil, nil, nil)
}

// ListOptions configures ListRepositories.
type ListOptions struct {
	Search string
	Sort   SortOrder
	Type   RepositoryType
	Limit  int
	Offset int
}

// SortOrder is the sort parameter accepted by the API.
type SortOrder string

const (
	SortNameAsc       SortOrder = "name_asc"
	SortNameDesc      SortOrder = "name_desc"
	SortUpdatedAtAsc  SortOrder = "updated_at_asc"
	SortUpdatedAtDesc SortOrder = "updated_at_desc"
)

// RepositoryType is the optional type filter.
type RepositoryType string

const (
	TypeGit     RepositoryType = "git"
	TypeModel   RepositoryType = "model"
	TypeDataset RepositoryType = "dataset"
)
