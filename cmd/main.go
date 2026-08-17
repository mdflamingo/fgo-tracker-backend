package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/mdflamingo/fgo-tracker-backend/internal/config"
	"github.com/mdflamingo/fgo-tracker-backend/internal/handler"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"github.com/mdflamingo/fgo-tracker-backend/internal/repository"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// @title Online Subscription API
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

	logger.Log.Info("Running server", zap.String("address", conf.RunAddr))

	storage, err := initStorage(conf)
	if err != nil {
		logger.Log.Fatal("Failed to create storage", zap.Error(err))
	}
	defer storage.Close()

	r := handler.NewRouter(conf, storage)

	return http.ListenAndServe(conf.RunAddr, r)
}

func initStorage(conf *config.Config) (*repository.DBStorage, error) {
	if conf.DataBaseDSN == "" {
		return nil, errors.New("DATABASE_DSN is required")
	}

	logger.Log.Info("Attempting to use database storage", zap.String("dsn", conf.DataBaseDSN))
	storage, err := repository.NewDBStorage(conf.DataBaseDSN)
	if err != nil {
		logger.Log.Warn("Failed to initialize database storage", zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Successfully initialized database storage")
	return storage, nil
}
