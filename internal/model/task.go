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
	Name        string       `json:"name" example:"Task name" description:"Task name" validate:"required,max=255"`
	Description string       `json:"description" example:"Task description" description:"Task description" validate:"omitempty,max=1000"`
	Status      TaskStatus   `json:"status" example:"backlog" description:"Task status" validate:"omitempty,oneof=backlog in_progress review done"`
	Priority    TaskPriority `json:"priority" example:"medium" description:"Task priority" validate:"omitempty,oneof=low medium high critical"`
	ProjectId   uuid.UUID    `json:"project_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"Project id" validate:"required,uuid"`
	Deadline    *time.Time   `json:"deadline" example:"2025-07-01T00:00:00Z" description:"Deadline date" validate:"omitempty,gt=0"`
	AssignedIds *[]uuid.UUID `json:"assigned_ids" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"User ids" validate:"omitempty,dive,uuid"`
	ReviewerIds *[]uuid.UUID `json:"reviewer_ids" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"User ids" validate:"omitempty,dive,uuid"`
}

type TaskUpdateRequest struct {
	Name        string       `json:"name" example:"Update user table" description:"Task name" validate:"required,max=255"`
	Description string       `json:"description" example:"Some description" description:"Task description" validate:"omitempty,max=1000"`
	Status      TaskStatus   `json:"status" example:"backlog" description:"Task status" validate:"omitempty,oneof=backlog in_progress review done"`
	Priority    TaskPriority `json:"priority" example:"medium" description:"Task priority" validate:"omitempty,oneof=low medium high critical"`
	ProjectId   uuid.UUID    `json:"project_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"Project id" validate:"required,uuid"`
	Deadline    *time.Time   `json:"deadline" example:"2025-07-01T00:00:00Z" description:"Deadline date" validate:"omitempty,gt=0"`
	CompletedAt *time.Time   `json:"completed_at" example:"2025-07-01T00:00:00Z" description:"Completed date" validate:"omitempty,gt=0"`
	AssignedIds *[]uuid.UUID `json:"assigned_ids" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" validate:"omitempty,dive,uuid"`
	ReviewerIds *[]uuid.UUID `json:"reviewer_ids" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" validate:"omitempty,dive,uuid"`
}

type TaskCreateResponse struct {
	ID uuid.UUID `json:"id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
}

type TaskResponse struct {
	Id          uuid.UUID    `json:"id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	Name        string       `json:"name" example:"Update user table"`
	Description string       `json:"description" example:"Some description"`
	Status      TaskStatus   `json:"status" example:"backlog"`
	Priority    TaskPriority `json:"priority" example:"medium"`
	Project     ProjectDB    `json:"project"`
	Creator     UserDB       `json:"creator"`
	Assignees   []UserDB     `json:"assignees"`
	Reviewers   []UserDB     `json:"reviewers"`
	Deadline    *time.Time   `json:"deadline" example:"2025-07-01T00:00:00Z"`
	Completed   *time.Time   `json:"completed" example:"2025-07-01T00:00:00Z"`
}

type TaskListResponse struct {
	Id          uuid.UUID    `json:"id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	Name        string       `json:"name" example:"Update user table"`
	Description string       `json:"description" example:"Some description"`
	Status      TaskStatus   `json:"status" example:"backlog"`
	Priority    TaskPriority `json:"priority" example:"medium"`
	ProjectName string       `json:"project_name"`
}

// / DB models ///
type TaskCreate struct {
	Id          uuid.UUID
	Name        string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	ProjectId   uuid.UUID
	Deadline    *time.Time
}

type TaskUpdate struct {
	Id          uuid.UUID
	Name        string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	ProjectId   uuid.UUID
	Deadline    *time.Time
	CompletedAt *time.Time
}

type TaskUserCreate struct {
	Id     uuid.UUID
	UserId uuid.UUID
	TaskId uuid.UUID
	Role   TaskRole
}

type TaskDB struct {
	Id          uuid.UUID
	Name        string
	Description string
	Status      TaskStatus
	Priority    TaskPriority
	ProjectId   uuid.UUID
	Deadline    time.Time
}
