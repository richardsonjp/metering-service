package storage

import "metering-service/internal/services/metering"

type Handler struct {
	meteringService metering.Service
	maxUploadBytes  int64
}

func NewHandler(meteringService metering.Service, maxUploadBytes int64) *Handler {
	return &Handler{
		meteringService: meteringService,
		maxUploadBytes:  maxUploadBytes,
	}
}
