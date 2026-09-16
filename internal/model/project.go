package model

import "github.com/google/uuid"

type ProjectRole string

const (
	ProjectOwner  ProjectRole = "owner"
	ProjectMember ProjectRole = "member"
	ProjectViewer ProjectRole = "viewer"
)

// API models
type ProjectListResponse struct {
	Id   uuid.UUID
	Name string
}

type ProjectCreateRequest struct {
	Name      string       `json:"name" example:"Project name"`
	MemberIds *[]uuid.UUID `json:"member_ids" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"User ids" validate:"omitempty,dive,uuid"`
	ViewerIds *[]uuid.UUID `json:"viewer_ids" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba" description:"User ids" validate:"omitempty,dive,uuid"`
}

type ProjectCreateResponse struct {
	ID uuid.UUID `json:"id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
}

// DB models
type ProjectDB struct {
	Id   uuid.UUID
	Name string
}

type ProjectUserCreate struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	ProjectId uuid.UUID
	Role      ProjectRole
}

type ProjectCreate struct {
	Id   uuid.UUID
	Name string
}
