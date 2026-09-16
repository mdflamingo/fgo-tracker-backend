package model

import "github.com/google/uuid"

type UserDB struct {
	Id       uuid.UUID
	Username string
	Email    string
}
