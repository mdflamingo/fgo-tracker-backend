package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	er "github.com/mdflamingo/fgo-tracker-backend/internal/error"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	pg "github.com/mdflamingo/fgo-tracker-backend/internal/repository/postgres"
	"github.com/mdflamingo/fgo-tracker-backend/internal/service"
	"github.com/mdflamingo/fgo-tracker-backend/internal/validator"
	"go.uber.org/zap"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// @Summary Get all tasks
// @Description Return all tasks in system
// @Tags Tasks
// @Produce json
// @Success 200 {array} model.TaskListResponse "Tasks"
// @Failure 500 {object} ResponseError "Internal Server Error"
// @Router /api/task/list [get]
func (h *TaskHandler) GetList(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.taskService.GetList()
	if err != nil {
		logger.Log.Error("handler: failed to get tasks", zap.Error(err))
		er.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, tasks)
}

// @Summary Get task by ID
// @Description Returns information about a specific task
// @Tags Tasks
// @Produce json
// @Param id path string true "UUID"
// @Success 200 {object} model.TaskResponse "Task found"
// @Failure 400 {object} ResponseError "Invalid task ID"
// @Failure 404 {object} ResponseError "Task not found"
// @Failure 500 {object} ResponseError "Internal Server Error"
// @Router /api/task/{id} [get]
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		er.ResponseWithError(w, r, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.taskService.GetTask(taskID)
	if err != nil {
		if errors.Is(err, pg.ErrTaskNotFound) {
			er.ResponseWithError(w, r, http.StatusNotFound, "Task not found")
			return
		}
		logger.Log.Error("handler: failed to get task", zap.Error(err))
		er.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, task)
}

// @Summary Create new task
// @Description Creates a new task with optional assignee and reviewer
// @Tags Tasks
// @Accept json
// @Produce json
// @Param request body model.TaskCreateRequest true "Task data"
// @Success 201 {object} model.TaskCreateResponse "Created task ID"
// @Failure 400 {object} ResponseError "Bad Request"
// @Failure 500 {object} ResponseError "Internal Server Error"
// @Router /api/task [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req model.TaskCreateRequest

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		er.ResponseWithError(w, r, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := validator.GlobalValidator.Struct(req); err != nil {
		er.ResponseWithValidationError(w, r, validator.FormatValidationError(err))
		return
	}

	creatorID := uuid.MustParse("01a06bf1-8749-7cd1-884b-689d6a59f9c7")

	resp, err := h.taskService.CreateTask(req, creatorID)
	if err != nil {
		logger.Log.Error("handler: failed to create task", zap.Error(err))
		er.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
}

// @Summary Task update
// @Description Updates information about the existing task
// @Tags Task
// @Accept json
// @Param id path string true "UUID"
// @Param request body model.TaskUpdateRequest true "Data for update"
// @Success 200
// @Failure 400 {object} ResponseError "Invalid request"
// @Failure 404 {object} ResponseError "Task not found"
// @Failure 500 {object} ResponseError "Internal Server Error"
// @Router /api/task/{id} [put]
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		er.ResponseWithError(w, r, http.StatusBadRequest, "Invalid task ID")
		return
	}

	var req model.TaskUpdateRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		er.ResponseWithError(w, r, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := validator.GlobalValidator.Struct(req); err != nil {
		er.ResponseWithValidationError(w, r, validator.FormatValidationError(err))
		return
	}

	err = h.taskService.UpdateTask(r.Context(), taskID, req)
	if err != nil {
		if errors.Is(err, pg.ErrTaskNotFound) {
			er.ResponseWithError(w, r, http.StatusNotFound, "Task not found")
			return
		}
		logger.Log.Error("handler: failed to update task", zap.Error(err))
		er.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusOK)
}

// @Summary Delete task
// @Description Delete task by ID
// @Tags Tasks
// @Param id path string true "UUID"
// @Success 200
// @Failure 400 {object} ResponseError "Invalid task ID"
// @Failure 404 {object} ResponseError "Task not found"
// @Failure 500 {object} ResponseError "Internal Server Error"
// @Router /api/task/{id} [delete]
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		er.ResponseWithError(w, r, http.StatusBadRequest, "Invalid task ID")
		return
	}

	err = h.taskService.DeleteTask(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, pg.ErrTaskNotFound) {
			er.ResponseWithError(w, r, http.StatusNotFound, "Task not found")
			return
		}
		logger.Log.Error("handler: failed to delete task", zap.Error(err))
		er.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusOK)
}

func parseTaskID(r *http.Request) (uuid.UUID, error) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		return uuid.Nil, errors.New("task ID is required")
	}
	return uuid.Parse(idStr)
}
