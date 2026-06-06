package routes

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/cmd/apiserver/app/store"
	"metering-service/internal/middleware"
	"metering-service/pkg/utils/api"
	"metering-service/pkg/utils/errors"
)

const defaultBodyLimit = 1 << 30 // equal to 1GB

func NewHTTPServer(app *store.App) *fiber.App {
	f := fiber.New(fiber.Config{
		AppName:   app.Config.System.AppName,
		BodyLimit: defaultBodyLimit,

		EnablePrintRoutes: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return errors.Respond(c, err)
		},
	})

	f.Use(middleware.AccessLog())
	f.Use(middleware.Recovery())

	f.Get("/health", health)

	initAPIMeteringRoute(f.Group("/api"), app)

	initStorageRoute(f, app)

	return f
}

func health(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(api.Base{Message: "ok"})
}
