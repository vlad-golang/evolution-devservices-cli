package workflowapi

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
