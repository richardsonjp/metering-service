package routes

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/cmd/apiserver/app/store"
)

func initStorageRoute(router fiber.Router, app *store.App) {
	router.Post("/upload", app.MeteringMiddleware, app.StorageHandler.Upload)
	router.Get("/storage", app.StorageHandler.GetStorage)
}
