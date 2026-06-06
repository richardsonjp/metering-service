package middleware_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metering-service/internal/middleware"
	"metering-service/pkg/utils/logs"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	logs.Init("debug")
	var buf bytes.Buffer
	logs.Log.SetOutput(&buf)
	t.Cleanup(func() { logs.Log.SetOutput(os.Stdout) })
	return &buf
}

func TestAccessLog_LogsRequestAndResponseBodies(t *testing.T) {
	buf := captureLogs(t)

	app := fiber.New()
	app.Use(middleware.AccessLog())
	app.Post("/echo", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewBufferString(`{"hello":"world"}`))
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	line := buf.String()
	assert.Contains(t, line, "POST /echo -> 201")
	assert.Contains(t, line, "client_ip=")
	assert.Contains(t, line, "request_body=")
	assert.Contains(t, line, `hello`, "request payload logged")
	assert.Contains(t, line, "response_body=")
	assert.Contains(t, line, `ok`, "response payload logged")
}

func TestAccessLog_SummarizesMultipart(t *testing.T) {
	buf := captureLogs(t)

	app := fiber.New()
	app.Use(middleware.AccessLog())
	app.Post("/upload", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusCreated) })

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "x.png")
	require.NoError(t, err)
	_, _ = fw.Write([]byte("\x89PNG\r\n\x1a\n binary-ish content"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set(fiber.HeaderContentType, w.FormDataContentType())
	_, err = app.Test(req, -1)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), `request_body="<multipart/form-data,`,
		"multipart body summarized, not dumped raw")
}

func TestAccessLog_LevelFollowsStatus(t *testing.T) {
	buf := captureLogs(t)

	app := fiber.New()
	app.Use(middleware.AccessLog())
	app.Get("/bad", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusBadRequest) })
	app.Get("/boom", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusInternalServerError) })

	_, _ = app.Test(httptest.NewRequest(http.MethodGet, "/bad", nil), -1)
	_, _ = app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil), -1)

	out := buf.String()
	assert.Contains(t, out, "WARN", "4xx logs at warn")
	assert.Contains(t, out, "ERROR", "5xx logs at error")
}
