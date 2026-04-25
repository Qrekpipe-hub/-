package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/taskservice/internal/model"
	"example.com/taskservice/internal/repository"
)

// ErrNotFound is re-exported so handlers don't import repository directly.
var ErrNotFound = repository.ErrNotFound

// TaskService contains all business logic for tasks.
type TaskService struct {
	repo *repository.TaskRepository
}

func NewTaskService(repo *repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (s *TaskService) Create(ctx context.Context, req model.CreateTaskRequest) (*model.Task, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, req)
}

func (s *TaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TaskService) List(ctx context.Context) ([]*model.Task, error) {
	return s.repo.List(ctx)
}

func (s *TaskService) Update(ctx context.Context, id string, req model.UpdateTaskRequest) (*model.Task, error) {
	if err := validateUpdateRequest(req); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, req)
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ── Recurrence ────────────────────────────────────────────────────────────────

// Occurrences computes the dates on which a recurring task fires within
// [from, to]. It does NOT write anything to the database.
func (s *TaskService) Occurrences(ctx context.Context, id, fromStr, toStr string) (*model.OccurrencesResponse, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.Recurrence == nil {
		return nil, &ValidationError{Msg: "task has no recurrence rule"}
	}

	from, to, err := parseDateRange(fromStr, toStr)
	if err != nil {
		return nil, err
	}

	dates := task.Recurrence.Occurrences(task.DueDate, from, to)
	strs := make([]string, len(dates))
	for i, d := range dates {
		strs[i] = d.Format("2006-01-02")
	}
	return &model.OccurrencesResponse{TaskID: id, Dates: strs}, nil
}

// GenerateInstances computes the occurrence dates for [from, to] and
// persists one child task per date. Returns all created tasks.
func (s *TaskService) GenerateInstances(ctx context.Context, id string, req model.GenerateRequest) ([]*model.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.Recurrence == nil {
		return nil, &ValidationError{Msg: "task has no recurrence rule"}
	}
	// A generated task cannot itself be used as a generator template —
	// that would create confusing "grandchild" tasks.
	if task.ParentID != nil {
		return nil, &ValidationError{Msg: "cannot generate instances from a generated task; use the original template"}
	}

	from, to, err := parseDateRange(req.From, req.To)
	if err != nil {
		return nil, err
	}

	dates := task.Recurrence.Occurrences(task.DueDate, from, to)
	if len(dates) == 0 {
		return []*model.Task{}, nil
	}

	strs := make([]string, len(dates))
	for i, d := range dates {
		strs[i] = d.Format("2006-01-02")
	}
	return s.repo.GenerateInstances(ctx, task, strs)
}

// ── Validation ────────────────────────────────────────────────────────────────

// ValidationError signals a client mistake (HTTP 400).
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

func validateCreateRequest(req model.CreateTaskRequest) error {
	if req.Title == "" {
		return &ValidationError{Msg: "title is required"}
	}
	if req.DueDate.IsZero() {
		return &ValidationError{Msg: "due_date is required"}
	}
	if req.Status == "" {
		return &ValidationError{Msg: "status is required"}
	}
	if !validStatus(req.Status) {
		return &ValidationError{Msg: fmt.Sprintf("invalid status %q; allowed: pending, in_progress, done", req.Status)}
	}
	if req.Recurrence != nil {
		if err := req.Recurrence.Validate(); err != nil {
			return &ValidationError{Msg: "recurrence: " + err.Error()}
		}
	}
	return nil
}

func validateUpdateRequest(req model.UpdateTaskRequest) error {
	if req.Title == "" {
		return &ValidationError{Msg: "title is required"}
	}
	if req.DueDate.IsZero() {
		return &ValidationError{Msg: "due_date is required"}
	}
	if !validStatus(req.Status) {
		return &ValidationError{Msg: fmt.Sprintf("invalid status %q; allowed: pending, in_progress, done", req.Status)}
	}
	if req.Recurrence != nil {
		if err := req.Recurrence.Validate(); err != nil {
			return &ValidationError{Msg: "recurrence: " + err.Error()}
		}
	}
	return nil
}

func validStatus(s model.Status) bool {
	switch s {
	case model.StatusPending, model.StatusInProgress, model.StatusDone:
		return true
	}
	return false
}

func parseDateRange(fromStr, toStr string) (from, to time.Time, err error) {
	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, &ValidationError{Msg: "from and to query parameters are required (YYYY-MM-DD)"}
	}
	from, err = model.ParseDate(fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, &ValidationError{Msg: fmt.Sprintf("invalid from date %q: must be YYYY-MM-DD", fromStr)}
	}
	to, err = model.ParseDate(toStr)
	if err != nil {
		return time.Time{}, time.Time{}, &ValidationError{Msg: fmt.Sprintf("invalid to date %q: must be YYYY-MM-DD", toStr)}
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, &ValidationError{Msg: "to must be >= from"}
	}
	// Sanity cap: refuse overly large windows to prevent accidental bulk inserts.
	if to.Sub(from).Hours() > 24*366*2 {
		return time.Time{}, time.Time{}, &ValidationError{Msg: "date range must not exceed 2 years"}
	}
	return from, to, nil
}

// ValidateDateRange is exported for unit tests.
func ValidateDateRange(from, to string) error {
	_, _, err := parseDateRange(from, to)
	return err
}
