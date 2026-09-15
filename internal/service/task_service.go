package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
// @Failure 500 {object} ResponseError "Internal Server Error"
// @Router /api/task/list [get]
func (s *TaskService) GetList(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.repo.GetList()
	if err != nil {
		logger.Log.Error("failed to get tasks", zap.Error(err))

		ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
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
func (s *TaskService) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid task ID", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := s.repo.GetOne(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Warn("task not found", zap.String("task_id", taskID.String()))
			ResponseWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}
		logger.Log.Error("failed to get task", zap.Error(err))
		ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, task)
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
func (s *TaskService) UpdateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	var inputUpdateTask model.TaskUpdateRequest
	if err := json.Unmarshal(body, &inputUpdateTask); err != nil {
		logger.Log.Error("invalid request body", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	taskID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Error("invalid task ID", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, "invalid task ID")
		return
	}

	existingTask, err := s.repo.GetOne(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Warn("task not found", zap.String("task_id", taskID.String()))
			ResponseWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}
		logger.Log.Error("failed to get task", zap.Error(err))
		ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	projectID := inputUpdateTask.ProjectId
	if projectID == uuid.Nil {
		projectID = existingTask.Project.Id
	}

	var updateUserTasks []model.TaskUserCreate

	if inputUpdateTask.AssignedIds == nil {
		for _, assignee := range existingTask.Assignees {
			updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
				Id:        mustNewUUIDV7(),
				UserId:    assignee.Id,
				ProjectId: projectID,
				Role:      model.TaskAssignee,
			})
		}
	} else {
		for _, assigneeID := range *inputUpdateTask.AssignedIds {
			if assigneeID != uuid.Nil {
				updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
					Id:        mustNewUUIDV7(),
					UserId:    assigneeID,
					ProjectId: projectID,
					Role:      model.TaskAssignee,
				})
			}
		}
	}

	if inputUpdateTask.ReviewerIds == nil {
		for _, reviewer := range existingTask.Reviewers {
			updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
				Id:        mustNewUUIDV7(),
				UserId:    reviewer.Id,
				ProjectId: projectID,
				Role:      model.TaskReviewer,
			})
		}
	} else {
		for _, reviewerID := range *inputUpdateTask.ReviewerIds {
			if reviewerID != uuid.Nil {
				updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
					Id:        mustNewUUIDV7(),
					UserId:    reviewerID,
					ProjectId: projectID,
					Role:      model.TaskReviewer,
				})
			}
		}
	}

	updateTask := model.TaskUpdate{
		Id:          taskID,
		Name:        inputUpdateTask.Name,
		Description: inputUpdateTask.Description,
		Status:      inputUpdateTask.Status,
		Priority:    inputUpdateTask.Priority,
		ProjectId:   projectID,
		Deadline:    inputUpdateTask.Deadline,
		CompletedAt: inputUpdateTask.CompletedAt,
	}

	err = s.repo.Update(taskID, updateTask, updateUserTasks)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Warn("task not found during update", zap.String("task_id", taskID.String()))
			ResponseWithError(w, r, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}
		logger.Log.Error("failed to update task", zap.Error(err))
		ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	render.Status(r, http.StatusOK)
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
func (s *TaskService) CreateTask(w http.ResponseWriter, r *http.Request) {
	// creatorID := mustNewUUIDV7()
	creatorID := uuid.MustParse("01a06bf1-8749-7cd1-884b-689d6a59f9c7") // поле будет доставваться из токена

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	var inputTask model.TaskCreateRequest
	if err := json.Unmarshal(body, &inputTask); err != nil {
		logger.Log.Warn("invalid request body", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, "invalid request body")
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
		logger.Log.Error("failed to create task", zap.Error(err))
		ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	resp := model.TaskCreateResponse{ID: createdTaskID}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, resp)
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
func (s *TaskService) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid task ID", zap.Error(err))
		ResponseWithError(w, r, http.StatusBadRequest, "invalid task ID")
		return
	}

	err = s.repo.Delete(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			logger.Log.Warn("task not found", zap.String("task_id", taskID.String()))
			ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		} else {
			logger.Log.Error("failed to delete task", zap.Error(err))
			ResponseWithError(w, r, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
	}

	render.Status(r, http.StatusOK)
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
	/// без uuid невозможно создать новую запись в бд
	uuid, err := uuid.NewV7()
	if err != nil {
		panic("critical: failed to generate UUID: " + err.Error())
	}
	return uuid
}
