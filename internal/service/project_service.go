package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	pg "github.com/mdflamingo/fgo-tracker-backend/internal/repository/postgres"
	"go.uber.org/zap"
)

type ProjectService struct {
	repo *pg.DBStorage
}

func NewProjectService(repo *pg.DBStorage) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) GetList() ([]model.ProjectListResponse, error) {
	projects, err := s.repo.GetProjectList()
	if err != nil {
		logger.Log.Error("failed to get projects list", zap.Error(err))
		return nil, fmt.Errorf("ProjectService.GetList: %w", err)
	}
	return projects, nil
}

func (s *ProjectService) CreateProject(req model.ProjectCreateRequest, creatorID uuid.UUID) (model.ProjectCreateResponse, error) {
	projectID := generateUUIDv7()

	createProject := model.ProjectCreate{
		Id:   projectID,
		Name: req.Name,
	}

	var createUserProjects []model.ProjectUserCreate

	createUserProjects = append(createUserProjects, model.ProjectUserCreate{
		Id:        generateUUIDv7(),
		UserId:    creatorID,
		ProjectId: projectID,
		Role:      model.ProjectOwner,
	})

	for _, memberID := range *req.MemberIds {
		if memberID != uuid.Nil {
			createUserProjects = append(createUserProjects, model.ProjectUserCreate{
				Id:        generateUUIDv7(),
				UserId:    memberID,
				ProjectId: projectID,
				Role:      model.ProjectMember,
			})
		}
	}

	for _, viewerID := range *req.ViewerIds {
		createUserProjects = append(createUserProjects, model.ProjectUserCreate{
			Id:        generateUUIDv7(),
			UserId:    viewerID,
			ProjectId: projectID,
			Role:      model.ProjectViewer,
		})

	}

	if err := s.repo.CreateProject(createProject, createUserProjects); err != nil {
		logger.Log.Error("failed to create project in repo", zap.Error(err))
		return model.ProjectCreateResponse{}, fmt.Errorf("ProjectService.CreateProject: %w", err)
	}

	return model.ProjectCreateResponse{ID: projectID}, nil
}
