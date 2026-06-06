package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metering-service/internal/meter"
	"metering-service/internal/middleware"
	"metering-service/internal/services/metering"
)

func TestMetering_TracksAndEnforcesCap(t *testing.T) {
	store := meter.NewStorage(1, 0)
	defer store.Close()
	svc := metering.NewService(store)

	app := fiber.New()
	handlerReached := 0
	app.Post("/x", middleware.Metering(svc), func(c *fiber.Ctx) error {
		handlerReached++
		res, ok := middleware.TrackResultFrom(c)
		require.True(t, ok, "track result must be present in Locals")
		return c.JSON(res)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/x", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest(http.MethodPost, "/x", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, 1, handlerReached, "handler must not run once the cap is hit")
}

func TestRecovery_PanicBecomes500WithoutLeak(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.Recovery())
	app.Get("/boom", func(c *fiber.Ctx) error {
		panic("super secret internal detail")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "INTERNAL_SERVER_ERROR")
	assert.NotContains(t, string(body), "super secret internal detail")
}
