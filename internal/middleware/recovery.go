package middleware

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/pkg/utils/errors"
	"metering-service/pkg/utils/logs"
)

func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logs.Panic(r, logs.Fields{"method": c.Method(), "path": c.Path()})
				err = errors.Respond(c, errors.From("INTERNAL_SERVER_ERROR"))
			}
		}()
		return c.Next()
	}
}
