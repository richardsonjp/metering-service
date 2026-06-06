package api_metering

import "metering-service/internal/services/metering"

type Handler struct {
	meteringService metering.Service
}

func NewHandler(meteringService metering.Service) *Handler {
	return &Handler{meteringService: meteringService}
}
