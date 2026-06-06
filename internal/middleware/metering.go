package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"metering-service/internal/services/metering"
	"metering-service/pkg/utils/errors"
)

const localTrackResult = "metering.track_result"

func Metering(svc metering.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		endpoint := strings.Clone(c.Path())
		res, err := svc.TrackRequest(c.Context(), endpoint)
		if err != nil {
			return errors.Respond(c, err)
		}
		c.Locals(localTrackResult, res)
		return c.Next()
	}
}

func TrackResultFrom(c *fiber.Ctx) (*metering.TrackResponse, bool) {
	res, ok := c.Locals(localTrackResult).(*metering.TrackResponse)
	return res, ok
}
