package handler

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	"github.com/mdflamingo/fgo-tracker-backend/internal/service"
	"github.com/mdflamingo/fgo-tracker-backend/internal/validator"
	"go.uber.org/zap"
)

type ProjectHandler struct {
	projectService *service.ProjectService
}

func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

// @Summary Get all projects
// @Description Return all projects in system
// @Tags Projects
// @Produce json
// @Success 200 {array} model.ProjectListResponse "Tasks"
// @Failure 500 {object} model.ResponseError "Internal Server Error"
// @Router /api/project/list [get]
func (h *ProjectHandler) GetList(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.projectService.GetList()
	if err != nil {
		logger.Log.Error("handler: failed to get projects", zap.Error(err))
		model.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, tasks)
}

// @Summary Create new projects
// @Description Creates a new project
// @Tags Projects
// @Accept json
// @Produce json
// @Param request body model.ProjectCreateRequest true "Project data"
// @Success 201 {object} model.ProjectCreateResponse "Created project ID"
// @Failure 400 {object} model.ResponseError "Bad Request"
// @Failure 500 {object} model.ResponseError "Internal Server Error"
// @Router /api/project [post]
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req model.ProjectCreateRequest

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		logger.Log.Error("handler: decode JSON error", zap.Error(err))
		model.ResponseWithError(w, r, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if err := validator.GlobalValidator.Struct(req); err != nil {
		logger.Log.Error("handler: request body validation error", zap.Error(err))
		model.ResponseWithValidationError(w, r, validator.FormatValidationError(err))
		return
	}

	creatorID := uuid.MustParse("01a06bf1-8749-7cd1-884b-689d6a59f9c7") // извлекать из токена

	response, err := h.projectService.CreateProject(req, creatorID)
	if err != nil {
		logger.Log.Error("handler: failed to create project", zap.Error(err))
		model.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}
