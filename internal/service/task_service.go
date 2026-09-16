package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	pg "github.com/mdflamingo/fgo-tracker-backend/internal/repository/postgres"
	"go.uber.org/zap"
)

type TaskService struct {
	repo *pg.DBStorage
}

func NewTaskService(repo *pg.DBStorage) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) GetList() ([]model.TaskListResponse, error) {
	tasks, err := s.repo.GetTaskList()
	if err != nil {
		logger.Log.Error("failed to get tasks list", zap.Error(err))
		return nil, fmt.Errorf("TaskService.GetList: %w", err)
	}
	return tasks, nil
}

func (s *TaskService) GetTask(taskID uuid.UUID) (*model.TaskResponse, error) {
	task, err := s.repo.GetOneTask(taskID)
	if err != nil {
		if errors.Is(err, pg.ErrTaskNotFound) {
			return nil, fmt.Errorf("TaskService.GetTask: %w", err)
		}
		logger.Log.Error("failed to get task", zap.String("task_id", taskID.String()), zap.Error(err))
		return nil, fmt.Errorf("TaskService.GetTask: %w", err)
	}
	return &task, nil
}

func (s *TaskService) CreateTask(req model.TaskCreateRequest, creatorID uuid.UUID) (model.TaskCreateResponse, error) {
	taskID := generateUUIDv7()

	createTask := model.TaskCreate{
		Id:          taskID,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		ProjectId:   req.ProjectId,
		Deadline:    req.Deadline,
	}

	var createUserTasks []model.TaskUserCreate

	createUserTasks = append(createUserTasks, model.TaskUserCreate{
		Id:     generateUUIDv7(),
		UserId: creatorID,
		TaskId: taskID,
		Role:   model.TaskCreator,
	})

	for _, assigneeID := range *req.AssignedIds {
		if assigneeID != uuid.Nil {
			createUserTasks = append(createUserTasks, model.TaskUserCreate{
				Id:     generateUUIDv7(),
				UserId: assigneeID,
				TaskId: taskID,
				Role:   model.TaskAssignee,
			})
		}
	}

	for _, reviewerID := range *req.ReviewerIds {
		createUserTasks = append(createUserTasks, model.TaskUserCreate{
			Id:     generateUUIDv7(),
			UserId: reviewerID,
			TaskId: taskID,
			Role:   model.TaskReviewer,
		})

	}

	if err := s.repo.CreateTask(createTask, createUserTasks); err != nil {
		logger.Log.Error("failed to create task in repo", zap.Error(err))
		return model.TaskCreateResponse{}, fmt.Errorf("TaskService.CreateTask: %w", err)
	}

	return model.TaskCreateResponse{ID: taskID}, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, taskID uuid.UUID, req model.TaskUpdateRequest) error {
	existingTask, err := s.repo.GetOneTask(taskID)
	if err != nil {
		if errors.Is(err, pg.ErrTaskNotFound) {
			return fmt.Errorf("TaskService.UpdateTask: %w", pg.ErrTaskNotFound)
		}
		return fmt.Errorf("TaskService.UpdateTask (get existing): %w", err)
	}

	projectID := req.ProjectId
	if projectID == uuid.Nil {
		projectID = existingTask.Project.Id
	}

	var updateUserTasks []model.TaskUserCreate

	if req.AssignedIds == nil {
		for _, assignee := range existingTask.Assignees {
			updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
				Id:     generateUUIDv7(),
				UserId: assignee.Id,
				TaskId: taskID,
				Role:   model.TaskAssignee,
			})
		}
	} else {
		for _, assigneeID := range *req.AssignedIds {
			if assigneeID != uuid.Nil {
				updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
					Id:     generateUUIDv7(),
					UserId: assigneeID,
					TaskId: taskID,
					Role:   model.TaskAssignee,
				})
			}
		}
	}

	if req.ReviewerIds == nil {
		for _, reviewer := range existingTask.Reviewers {
			updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
				Id:     generateUUIDv7(),
				UserId: reviewer.Id,
				TaskId: taskID,
				Role:   model.TaskReviewer,
			})
		}
	} else {
		for _, reviewerID := range *req.ReviewerIds {
			if reviewerID != uuid.Nil {
				updateUserTasks = append(updateUserTasks, model.TaskUserCreate{
					Id:     generateUUIDv7(),
					UserId: reviewerID,
					TaskId: taskID,
					Role:   model.TaskReviewer,
				})
			}
		}
	}

	updateTask := model.TaskUpdate{
		Id:          taskID,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		ProjectId:   projectID,
		Deadline:    req.Deadline,
		CompletedAt: req.CompletedAt,
	}

	if err := s.repo.UpdateTask(taskID, updateTask, updateUserTasks); err != nil {
		logger.Log.Error("failed to update task in repo", zap.String("task_id", taskID.String()), zap.Error(err))
		return fmt.Errorf("TaskService.UpdateTask: %w", err)
	}

	return nil
}

func (s *TaskService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	if err := s.repo.DeleteTask(taskID); err != nil {
		if errors.Is(err, pg.ErrTaskNotFound) {
			return fmt.Errorf("TaskService.DeleteTask: %w", pg.ErrTaskNotFound)
		}
		logger.Log.Error("failed to delete task", zap.String("task_id", taskID.String()), zap.Error(err))
		return fmt.Errorf("TaskService.DeleteTask: %w", err)
	}
	return nil
}
