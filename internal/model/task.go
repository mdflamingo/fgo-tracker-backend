package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string
type TaskPriority string
type TaskRole string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
)

const (
	TaskPriorityLow      TaskPriority = "low"
	TaskPriorityMedium   TaskPriority = "medium"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityCritical TaskPriority = "critical"
)

const (
	TaskCreator  TaskRole = "creator"
	TaskAssignee TaskRole = "assignee"
	TaskReviewer TaskRole = "reviewer"
)

// / API models ///
// нужно добавить возможность принимать файлы, ссылки и тд
type TaskCreateRequest struct {
	Name        string       `json:"name" example:"Update user table" description:"Task name" validate:"required"`
	Description string       `json:"description" example:"Some description" description:"Task description" validate:"required"`
	Status      TaskStatus   `json:"status" example:"backlog" description:"Task status" validate:"required"`
	Priority    TaskPriority `json:"priority" example:"medium" description:"Task priority" validate:"required"`
	ProjectId   uuid.UUID    `json:"project_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"Project id" validate:"required"`
	AssignedId  uuid.UUID    `json:"assigned_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"User id with role assignee" validate:"required"`
	ReviewerId  uuid.UUID    `json:"reviewer_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"User id with role reviewer" validate:"required"`
	Deadline    time.Time    `json:"deadline" example:"2025-07-01T00:00:00Z" description:"Deadline date" validate:"required"`
}

type TaskCreateResponse struct {
	ID uuid.UUID `json:"id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"Created task ID" validate:"required"`
}

// / DB models ///
type TaskCreate struct {
	Id          uuid.UUID
	Name        string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	ProjectId   uuid.UUID
	Deadline    time.Time
}

type TaskUserCreate struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	ProjectId uuid.UUID
	Role      TaskRole
}
