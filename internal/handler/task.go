package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"example.com/taskservice/internal/model"
	"example.com/taskservice/internal/service"
	"github.com/gorilla/mux"
)

// TaskHandler wires HTTP requests to the service layer.
type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// ── Register routes ───────────────────────────────────────────────────────────

func (h *TaskHandler) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// Core CRUD
	api.HandleFunc("/tasks", h.Create).Methods(http.MethodPost)
	api.HandleFunc("/tasks", h.List).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id}", h.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id}", h.Update).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id}", h.Delete).Methods(http.MethodDelete)

	// Recurrence
	api.HandleFunc("/tasks/{id}/occurrences", h.Occurrences).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id}/generate", h.GenerateInstances).Methods(http.MethodPost)
}

// ── Core CRUD ─────────────────────────────────────────────────────────────────

// Create godoc
// @Summary     Create a task
// @Tags        tasks
// @Accept      json
// @Produce     json
// @Param       body body model.CreateTaskRequest true "Task data"
// @Success     201  {object} model.Task
// @Failure     400  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /tasks [post]
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	task, err := h.svc.Create(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, task)
}

// List godoc
// @Summary     List all tasks
// @Tags        tasks
// @Produce     json
// @Success     200 {array}  model.Task
// @Failure     500 {object} errorResponse
// @Router      /tasks [get]
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	if tasks == nil {
		tasks = []*model.Task{}
	}
	respondJSON(w, http.StatusOK, tasks)
}

// GetByID godoc
// @Summary     Get a task by ID
// @Tags        tasks
// @Produce     json
// @Param       id   path     string true "Task UUID"
// @Success     200  {object} model.Task
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /tasks/{id} [get]
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	task, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, task)
}

// Update godoc
// @Summary     Update a task
// @Tags        tasks
// @Accept      json
// @Produce     json
// @Param       id   path     string true "Task UUID"
// @Param       body body     model.UpdateTaskRequest true "Updated task data"
// @Success     200  {object} model.Task
// @Failure     400  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /tasks/{id} [put]
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	task, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		handleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, task)
}

// Delete godoc
// @Summary     Delete a task
// @Tags        tasks
// @Param       id   path string true "Task UUID"
// @Success     204
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /tasks/{id} [delete]
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Recurrence ────────────────────────────────────────────────────────────────

// Occurrences godoc
// @Summary     Preview occurrence dates for a recurring task
// @Description Returns the list of dates (no DB writes) on which the task
//
//	would fire within the given range.
//
// @Tags        tasks
// @Produce     json
// @Param       id   path     string true  "Task UUID"
// @Param       from query    string true  "Start date YYYY-MM-DD"
// @Param       to   query    string true  "End date   YYYY-MM-DD"
// @Success     200  {object} model.OccurrencesResponse
// @Failure     400  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /tasks/{id}/occurrences [get]
func (h *TaskHandler) Occurrences(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	resp, err := h.svc.Occurrences(r.Context(), id, from, to)
	if err != nil {
		handleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, resp)
}

// GenerateInstances godoc
// @Summary     Generate task instances for a date range
// @Description Creates one child task per occurrence date within [from, to].
//
//	The parent template task is left unchanged. Generated tasks
//	start with status "pending" and have parent_id set.
//
// @Tags        tasks
// @Accept      json
// @Produce     json
// @Param       id   path     string true "Task UUID (template)"
// @Param       body body     model.GenerateRequest true "Date range"
// @Success     201  {array}  model.Task
// @Failure     400  {object} errorResponse
// @Failure     404  {object} errorResponse
// @Failure     500  {object} errorResponse
// @Router      /tasks/{id}/generate [post]
func (h *TaskHandler) GenerateInstances(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req model.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	tasks, err := h.svc.GenerateInstances(r.Context(), id, req)
	if err != nil {
		handleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, tasks)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type errorResponse struct {
	Error string `json:"error"`
}

func respondJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func respondError(w http.ResponseWriter, code int, msg string) {
	respondJSON(w, code, errorResponse{Error: msg})
}

func handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if service.IsValidationError(err) {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondError(w, http.StatusInternalServerError, "internal server error")
}
