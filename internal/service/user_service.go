package service

import (
	"fmt"

	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	pg "github.com/mdflamingo/fgo-tracker-backend/internal/repository/postgres"
	"go.uber.org/zap"
)

type UserService struct {
	repo *pg.DBStorage
}

func NewUserService(repo *pg.DBStorage) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetList() ([]model.UserDB, error) {
	users, err := s.repo.GetUserList()
	if err != nil {
		logger.Log.Error("failed to get users list", zap.Error(err))
		return nil, fmt.Errorf("UserService.GetList: %w", err)
	}
	return users, nil
}
