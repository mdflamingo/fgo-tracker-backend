package handler

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/model"
	"github.com/mdflamingo/fgo-tracker-backend/internal/service"
	"go.uber.org/zap"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// @Summary Get all users
// @Description Return all users in system
// @Tags Users
// @Produce json
// @Success 200 {array} model.UserDB "Users"
// @Failure 500 {object} model.ResponseError "Internal Server Error"
// @Router /api/user/list [get]
func (h *UserHandler) GetList(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.GetList()
	if err != nil {
		logger.Log.Error("handler: failed to get users", zap.Error(err))
		model.ResponseWithError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, users)
}
