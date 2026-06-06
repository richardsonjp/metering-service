package errors_test

import (
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metering-service/pkg/utils/errors"
)

func TestFrom_KnownCode(t *testing.T) {
	e := errors.From("REQUEST_LIMIT_EXCEEDED")
	assert.Equal(t, "REQUEST_LIMIT_EXCEEDED", e.Code)
	assert.Equal(t, http.StatusTooManyRequests, e.Status)
}

func TestFrom_UnknownCodeFallsBackTo500(t *testing.T) {
	e := errors.From("NOPE_NOT_REGISTERED")
	assert.Equal(t, "INTERNAL_SERVER_ERROR", e.Code)
	assert.Equal(t, http.StatusInternalServerError, e.Status)
}

func TestNewAndBuilders(t *testing.T) {
	e := errors.NewAppError("CUSTOM", 418, "teapot").
		WithMessage("i am a teapot").
		WithStatus(419).
		WithDetail("d1").
		WithDetails([]string{"d2", "d3"})

	assert.Equal(t, "CUSTOM", e.Code)
	assert.Equal(t, 419, e.Status)
	assert.Equal(t, "i am a teapot", e.Message)
	assert.Equal(t, []string{"d1", "d2", "d3"}, e.Details)
	assert.Equal(t, "CUSTOM - i am a teapot", e.Error())
}

func TestIs(t *testing.T) {
	err := errors.From("FILE_REQUIRED")
	assert.True(t, errors.Is(err, "FILE_REQUIRED"))
	assert.False(t, errors.Is(err, "BAD_REQUEST"))
	assert.False(t, errors.Is(stderrors.New("plain"), "FILE_REQUIRED"))
}

func TestRespond(t *testing.T) {
	app := fiber.New()
	app.Get("/app-error", func(c *fiber.Ctx) error {
		return errors.Respond(c, errors.From("BAD_REQUEST").WithDetail("nope"))
	})
	app.Get("/fiber-error", func(c *fiber.Ctx) error {
		return errors.Respond(c, fiber.NewError(fiber.StatusRequestEntityTooLarge, "too big"))
	})
	app.Get("/unknown-error", func(c *fiber.Ctx) error {
		return errors.Respond(c, stderrors.New("boom"))
	})
	app.Get("/nil-error", func(c *fiber.Ctx) error {
		return errors.Respond(c, nil)
	})

	tests := []struct {
		path     string
		wantCode int
		wantBody string
	}{
		{"/app-error", http.StatusBadRequest, "BAD_REQUEST"},
		{"/fiber-error", http.StatusRequestEntityTooLarge, "REQUEST_ENTITY_TOO_LARGE"},
		{"/unknown-error", http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil), -1)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), tt.wantBody)
		})
	}

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/nil-error", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
