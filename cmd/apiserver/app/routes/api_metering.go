package routes

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/cmd/apiserver/app/store"
)

func initAPIMeteringRoute(group fiber.Router, app *store.App) {
	group.Post("/endpoint1", app.MeteringMiddleware, app.APIMeteringHandler.TrackEndpoint)
	group.Get("/metrics", app.APIMeteringHandler.GetMetrics)
}
