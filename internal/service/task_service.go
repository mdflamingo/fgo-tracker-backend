package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	"github.com/mdflamingo/fgo-tracker-backend/internal/repository"

	"go.uber.org/zap"
)

type TaskService struct {
	repo *repository.DBStorage
}

func NewTaskService(repo *repository.DBStorage) *TaskService {
	return &TaskService{repo: repo}
}

// @Summary Get all tasks
// @Description Return all tasks in system
// @Tags Tasks
// @Produce json
// @Success 200 {array} model.TaskListResponse "Tasks"
// @Failure 204 {string} string "Tasks not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/task/list [get]
func (s *TaskService) GetList(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.repo.GetList()
	if err != nil {
		logger.Log.Error("failed to get subscriptions", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(tasks) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	respJSON, err := json.Marshal(tasks)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

// @Summary Get task by ID
// @Description Returns information about a specific task
// @Tags Tasks
// @Produce json
// @Param id path string true "UUID"
// @Success 200 {object} model.TaskResponse "Task found"
// @Failure 400 {string} string "Invalid task ID"
// @Failure 404 {string} string "Task not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/task/{id} [get]
func (s *TaskService) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid task ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := s.repo.GetOne(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Error("task not found", zap.Error(err))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Log.Error("failed to get task", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(task)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

// @Summary Task update
// @Description Updates information about the existing task
// @Tags Task
// @Accept json
// @Produce json
// @Param id path string true "UUID"
// @Param request body model.TaskUpdateRequest true "Data for update"
// @Success 200 {string} string "Successful task update"s
// @Failure 400 {string} string "Invalid task ID"
// @Failure 404 {string} string "Task not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/task/{id} [put]
func (s *TaskService) UpdateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var updateTask model.TaskUpdateRequest
	if err := json.Unmarshal(body, &updateTask); err != nil {
		logger.Log.Warn("invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	taskID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid task ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.repo.Update(taskID, updateTask)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Error("task not found", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			logger.Log.Error("failed to update task", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// @Summary Create new task
// @Description Creates a new task with optional assignee and reviewer
// @Tags Tasks
// @Accept json
// @Produce json
// @Param request body model.TaskCreateRequest true "Task data"
// @Success 201 {object} model.TaskCreateResponse "Created task ID"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/task [post]
func (s *TaskService) CreateTask(w http.ResponseWriter, r *http.Request) {
	// creatorID := mustNewUUIDV7()
	creatorID := uuid.MustParse("01a06bf1-8749-7cd1-884b-689d6a59f9c7") // поле будет доставваться из токена

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var inputTask model.TaskCreateRequest
	if err := json.Unmarshal(body, &inputTask); err != nil {
		logger.Log.Warn("invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdTaskID := mustNewUUIDV7()
	createTask := model.TaskCreate{
		Id:          createdTaskID,
		Name:        inputTask.Name,
		Description: inputTask.Description,
		Status:      inputTask.Status,
		Priority:    inputTask.Priority,
		ProjectId:   inputTask.ProjectId,
		Deadline:    inputTask.Deadline,
	}
	var createUserTasks []model.TaskUserCreate

	if inputTask.AssignedId != uuid.Nil {
		createUserTasks = append(
			createUserTasks,
			model.TaskUserCreate{
				Id:        mustNewUUIDV7(),
				UserId:    inputTask.AssignedId,
				ProjectId: inputTask.ProjectId,
				Role:      model.TaskAssignee,
			},
		)
	}

	if inputTask.ReviewerId != uuid.Nil {
		createUserTasks = append(
			createUserTasks,
			model.TaskUserCreate{
				Id:        mustNewUUIDV7(),
				UserId:    inputTask.ReviewerId,
				ProjectId: inputTask.ProjectId,
				Role:      model.TaskReviewer,
			},
		)
	}

	createUserTasks = append(
		createUserTasks,
		model.TaskUserCreate{
			Id:        mustNewUUIDV7(),
			UserId:    creatorID,
			ProjectId: inputTask.ProjectId,
			Role:      model.TaskCreator,
		},
	)

	err = s.repo.Create(createTask, createUserTasks)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := model.TaskCreateResponse{ID: createdTaskID}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		logger.Log.Error("failed to marshal response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(respJSON)
}

// @Summary Delete task
// @Description Delete task by ID
// @Tags Tasks
// @Param id path string true "UUID"
// @Success 200 {string} string "Successful task deletion"
// @Failure 400 {string} string "Invalid task ID"
// @Failure 404 {string} string "Task not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/task/{id} [delete]
func (s *TaskService) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid task ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.repo.Delete(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Error("task not found", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else {
			logger.Log.Error("failed to delete task", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func parseTaskID(r *http.Request) (uuid.UUID, error) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		return uuid.Nil, errors.New("task ID is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid task ID: %w", err)
	}

	return id, nil
}

func mustNewUUIDV7() uuid.UUID {
	///Здесь мы генерируем UUID. Если это не удалось, значит, мир рушится, и нам лучше немедленно упасть, чем продолжать работать с некорректными данными.
	uuid, err := uuid.NewV7()
	if err != nil {
		panic("critical: failed to generate UUID: " + err.Error())
	}
	fmt.Println(uuid)
	return uuid
}
