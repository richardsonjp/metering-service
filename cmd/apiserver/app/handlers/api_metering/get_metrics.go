package api_metering

import (
	"github.com/gofiber/fiber/v2"

	"metering-service/pkg/utils/api"
	"metering-service/pkg/utils/errors"
)

func (h *Handler) GetMetrics(c *fiber.Ctx) error {
	res, err := h.meteringService.Metrics(c.Context())
	if err != nil {
		return errors.Respond(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(api.Base{Data: res})
}
