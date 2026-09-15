package handler

import (
	"net/http"

	_ "github.com/mdflamingo/fgo-tracker-backend/api/swagger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mdflamingo/fgo-tracker-backend/internal/config"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	pg "github.com/mdflamingo/fgo-tracker-backend/internal/repository/postgres"
	"github.com/mdflamingo/fgo-tracker-backend/internal/service"
	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter(conf *config.Config, storage *pg.DBStorage) *chi.Mux {
	r := chi.NewRouter()

	taskService := service.NewTaskService(storage)

	r.Use(middleware.Recoverer)
	r.Use(logger.RequestLogger)

	// // Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.UIConfig(map[string]string{
			"persistAuthorization": "true",
		}),
	))

	r.Group(func(r chi.Router) {
		r.Post("/api/task", func(w http.ResponseWriter, r *http.Request) {
			taskService.CreateTask(w, r)
		})
		r.Get("/api/task/list", func(w http.ResponseWriter, r *http.Request) {
			taskService.GetList(w, r)
		})
		r.Get("/api/task/{id}", func(w http.ResponseWriter, r *http.Request) {
			taskService.GetTask(w, r)
		})
		r.Put("/api/task/{id}", func(w http.ResponseWriter, r *http.Request) {
			taskService.UpdateTask(w, r)
		})
		r.Delete("/api/task/{id}", func(w http.ResponseWriter, r *http.Request) {
			taskService.DeleteTask(w, r)
		})
	})

	return r
}
