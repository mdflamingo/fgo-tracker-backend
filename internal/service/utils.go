package service

import (
	"github.com/google/uuid"
	"github.com/mdflamingo/fgo-tracker-backend/internal/logger"
	"go.uber.org/zap"
)

func generateUUIDv7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Log.Error("failed to generate UUID v7, falling back to v4", zap.Error(err))
		return uuid.New()
	}
	return id
}
