package store

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/cmd/apiserver/app/handlers/api_metering"
	"metering-service/cmd/apiserver/app/handlers/storage"
	"metering-service/config"
	"metering-service/internal/meter"
	"metering-service/internal/middleware"
	"metering-service/internal/services/metering"
)

type App struct {
	Config config.Config

	meterStore *meter.Storage

	APIMeteringHandler *api_metering.Handler
	StorageHandler     *storage.Handler
	MeteringMiddleware fiber.Handler
}

func Init(cfg config.Config) *App {
	meterStore := meter.NewStorage(cfg.Metering.RequestLimit, cfg.Metering.StorageLimit)
	meteringService := metering.NewService(meterStore)

	return &App{
		Config:             cfg,
		meterStore:         meterStore,
		APIMeteringHandler: api_metering.NewHandler(meteringService),
		StorageHandler:     storage.NewHandler(meteringService, cfg.Metering.MaxUploadBytes),
		MeteringMiddleware: middleware.Metering(meteringService),
	}
}

func (a *App) Shutdown() {
	a.meterStore.Close()
}
