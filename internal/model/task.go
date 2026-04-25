package model

import "time"

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// Task is the core domain object.
type Task struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	DueDate     time.Time   `json:"due_date"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`

	// Recurrence rule — non-nil only on template tasks.
	Recurrence *Recurrence `json:"recurrence,omitempty"`

	// ParentID is set on auto-generated task instances.
	ParentID *string `json:"parent_id,omitempty"`
}

// CreateTaskRequest is the body for POST /api/v1/tasks.
type CreateTaskRequest struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	DueDate     time.Time   `json:"due_date"`
	Recurrence  *Recurrence `json:"recurrence,omitempty"`
}

// UpdateTaskRequest is the body for PUT /api/v1/tasks/{id}.
type UpdateTaskRequest struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	DueDate     time.Time   `json:"due_date"`
	Recurrence  *Recurrence `json:"recurrence,omitempty"`
}

// GenerateRequest is the body for POST /api/v1/tasks/{id}/generate.
type GenerateRequest struct {
	From string `json:"from"` // "YYYY-MM-DD"
	To   string `json:"to"`   // "YYYY-MM-DD"
}

// OccurrencesResponse is returned by GET /api/v1/tasks/{id}/occurrences.
type OccurrencesResponse struct {
	TaskID string   `json:"task_id"`
	Dates  []string `json:"dates"` // "YYYY-MM-DD"
}

// ParseDate parses a "YYYY-MM-DD" string into midnight UTC.
func ParseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
