package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/mdflamingo/fgo-tracker-backend/internal/config"
	"github.com/mdflamingo/fgo-tracker-backend/internal/handler"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	pg "github.com/mdflamingo/fgo-tracker-backend/internal/repository/postgres"
	"github.com/mdflamingo/fgo-tracker-backend/internal/validator"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// @title Task Tracker API
// @version 1.0.0
// @description API for task tracker
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@example.com
// @host localhost:8080
// @BasePath /
func main() {
	_ = godotenv.Load()

	conf := config.ParseFlags()
	if err := run(conf); err != nil {
		log.Fatal(err)
	}
}

func run(conf *config.Config) error {
	if err := logger.Initialize(conf.LogLevel); err != nil {
		return err
	}

	validator.Init()

	logger.Log.Info("Running server", zap.String("address", conf.RunAddr))

	storage, err := pg.InitStorage(conf)
	if err != nil {
		logger.Log.Fatal("Failed to create storage", zap.Error(err))
	}
	defer storage.Close()

	r := handler.NewRouter(conf, storage)

	return http.ListenAndServe(conf.RunAddr, r)
}
