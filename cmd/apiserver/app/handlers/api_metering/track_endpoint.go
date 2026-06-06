package api_metering

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/internal/middleware"
	"metering-service/pkg/utils/api"
	"metering-service/pkg/utils/errors"
)

func (h *Handler) TrackEndpoint(c *fiber.Ctx) error {
	res, ok := middleware.TrackResultFrom(c)
	if !ok {
		return errors.Respond(c, errors.From("INTERNAL_SERVER_ERROR").
			WithDetail("metering middleware did not run for this route"))
	}
	return c.Status(fiber.StatusOK).JSON(api.Base{
		Message: "request recorded",
		Data:    res,
	})
}
