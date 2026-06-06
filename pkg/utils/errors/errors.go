package errors

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"metering-service/pkg/utils/logs"
)

func Respond(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return c.Status(appErr.Status).JSON(appErr)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		mapped := &AppError{
			Code:    codeFromStatus(fiberErr.Code),
			Status:  fiberErr.Code,
			Message: fiberErr.Message,
		}
		return c.Status(mapped.Status).JSON(mapped)
	}

	logs.Error("unhandled error", logs.Fields{"error": err.Error()})
	internal := From("INTERNAL_SERVER_ERROR")
	return c.Status(internal.Status).JSON(internal)
}

func Is(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

func codeFromStatus(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "INTERNAL_SERVER_ERROR"
	}
	out := make([]rune, 0, len(text))
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r-('a'-'A'))
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
