package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

// @Summary Получение списка всех подписок
// @Description Возвращает список всех подписок в системе
// @Tags Subscriptions
// @Produce json
// @Success 200 {array} model.Subscription "Успешный ответ со списком подписок"
// @Failure 204 {string} string "Нет содержимого (список подписок пуст)"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/subscription/list [get]
func (s *TaskService) GetList(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := s.repo.GetList()
	if err != nil {
		logger.Log.Error("failed to get subscriptions", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(subscriptions) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	respJSON, err := json.Marshal(subscriptions)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

// @Summary Получение подписки по ID
// @Description Возвращает информацию о конкретной подписке
// @Tags Subscriptions
// @Produce json
// @Param id path int true "ID подписки" minimum(1)
// @Success 200 {object} model.Subscription "Успешный ответ с информацией о подписке"
// @Failure 204 {string} string "Подписка не найдена"
// @Failure 400 {string} string "Неверный ID подписки"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/subscription/{id} [get]
func (s *TaskService) GetTask(w http.ResponseWriter, r *http.Request) {
	subscriptionID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid subscription ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subscription, err := s.repo.GetOne(subscriptionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			logger.Log.Error("subscription not found", zap.Error(err))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		logger.Log.Error("failed to get subscription", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	respJSON, err := json.Marshal(subscription)
	if err != nil {
		logger.Log.Error("failed to marshal response to JSON", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

// @Summary Обновление подписки
// @Description Обновляет информацию о существующей подписке
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param id path int true "ID подписки для обновления" minimum(1)
// @Param request body model.Subscription true "Данные для обновления"
// @Success 200 {string} string "Подписка успешно обновлена"
// @Failure 204 {string} string "Подписка не найдена"
// @Failure 400 {string} string "Неверный запрос"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/subscription/{id} [put]
func (s *TaskService) UpdateTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var updateSubscription model.Subscription
	if err := json.Unmarshal(body, &updateSubscription); err != nil {
		logger.Log.Warn("invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	subscriptionID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid subscription ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.repo.Update(subscriptionID, updateSubscription)
	if err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			logger.Log.Error("subscription not found", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
			return
		} else {
			logger.Log.Error("failed to update subscription", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// @Summary Create new task
// @Description
// @Tags Tasks
// @Accept json
// @Produce json
// @Param request body model.TaskCreateRequest true ""
// @Success 201 {object} model.TaskCreateResponse ""
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/task [post]
func (s *TaskService) CreateTask(w http.ResponseWriter, r *http.Request) {
	creatorID := mustNewUUIDV7()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("failed to read request body", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var inputTask model.TaskCreateRequest
	if err := json.Unmarshal(body, &inputTask); err != nil {
		logger.Log.Warn("invalid request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		logger.Log.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := model.TaskCreateResponse{ID: createdTaskID}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		logger.Log.Error("failed to marshal response", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(respJSON)
}

// @Summary Delete task
// @Description Delete task by ID
// @Tags Tasks
// @Param id path int true "ID task for delete" minimum(1)
// @Success 200 {string} string ""
// @Failure 204 {string} string ""
// @Failure 400 {string} string ""
// @Failure 500 {string} string ""
// @Router /api/subscription/{id} [delete]
func (s *TaskService) DeleteTask(w http.ResponseWriter, r *http.Request) {
	subscriptionID, err := parseTaskID(r)
	if err != nil {
		logger.Log.Warn("invalid subscription ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.repo.Delete(subscriptionID)
	if err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			logger.Log.Error("subscription not found", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
			return
		} else {
			logger.Log.Error("failed to delete subscription", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func parseTaskID(r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		return 0, errors.New("subscription ID is required")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid subscription ID")
	}
	return id, nil
}

func mustNewUUIDV7() uuid.UUID {
	///Здесь мы генерируем UUID. Если это не удалось, значит, мир рушится, и нам лучше немедленно упасть, чем продолжать работать с некорректными данными.
	uuid, err := uuid.NewV7()
	if err != nil {
		panic("critical: failed to generate UUID: " + err.Error())
	}

	return uuid
}
