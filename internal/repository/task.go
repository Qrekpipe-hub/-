package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"example.com/taskservice/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("task not found")

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) Create(ctx context.Context, req model.CreateTaskRequest) (*model.Task, error) {
	recJSON, err := marshalRecurrence(req.Recurrence)
	if err != nil {
		return nil, fmt.Errorf("marshal recurrence: %w", err)
	}
	const q = `
		INSERT INTO tasks (title, description, status, due_date, recurrence)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, status, due_date, recurrence, parent_id, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, req.Title, req.Description, req.Status, req.DueDate, recJSON)
	return scanTask(row)
}

func (r *TaskRepository) createInstance(ctx context.Context, tx pgx.Tx, t model.Task) (*model.Task, error) {
	const q = `
		INSERT INTO tasks (title, description, status, due_date, parent_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, status, due_date, recurrence, parent_id, created_at, updated_at`
	row := tx.QueryRow(ctx, q, t.Title, t.Description, t.Status, t.DueDate, t.ParentID)
	return scanTask(row)
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*model.Task, error) {
	const q = `
		SELECT id, title, description, status, due_date, recurrence, parent_id, created_at, updated_at
		FROM tasks WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *TaskRepository) List(ctx context.Context) ([]*model.Task, error) {
	const q = `
		SELECT id, title, description, status, due_date, recurrence, parent_id, created_at, updated_at
		FROM tasks ORDER BY due_date, created_at`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) Update(ctx context.Context, id string, req model.UpdateTaskRequest) (*model.Task, error) {
	recJSON, err := marshalRecurrence(req.Recurrence)
	if err != nil {
		return nil, fmt.Errorf("marshal recurrence: %w", err)
	}
	const q = `
		UPDATE tasks
		SET title=$1, description=$2, status=$3, due_date=$4, recurrence=$5, updated_at=NOW()
		WHERE id=$6
		RETURNING id, title, description, status, due_date, recurrence, parent_id, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, req.Title, req.Description, req.Status, req.DueDate, recJSON, id)
	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM tasks WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GenerateInstances creates one task record per date, all linked to parent.
// Runs in a single transaction — all-or-nothing.
func (r *TaskRepository) GenerateInstances(ctx context.Context, parent *model.Task, dates []string) ([]*model.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	parentID := parent.ID
	var created []*model.Task

	for _, d := range dates {
		due, err := model.ParseDate(d)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", d, err)
		}
		instance := model.Task{
			Title:       parent.Title,
			Description: parent.Description,
			Status:      model.StatusPending,
			DueDate:     due,
			ParentID:    &parentID,
		}
		task, err := r.createInstance(ctx, tx, instance)
		if err != nil {
			return nil, fmt.Errorf("insert instance for %s: %w", d, err)
		}
		created = append(created, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (*model.Task, error) {
	var t model.Task
	var recRaw []byte
	var parentID *string

	err := s.Scan(&t.ID, &t.Title, &t.Description, &t.Status,
		&t.DueDate, &recRaw, &parentID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(recRaw) > 0 {
		var rec model.Recurrence
		if err := json.Unmarshal(recRaw, &rec); err != nil {
			return nil, fmt.Errorf("unmarshal recurrence: %w", err)
		}
		t.Recurrence = &rec
	}
	t.ParentID = parentID
	return &t, nil
}

func marshalRecurrence(r *model.Recurrence) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return json.Marshal(r)
}
