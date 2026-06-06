package middleware

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"metering-service/pkg/utils/logs"
)

const maxBodyLog = 2048

func AccessLog() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		reqBody := summarizeRequestBody(c)

		err := c.Next()

		status := c.Response().StatusCode()
		fields := logs.Fields{
			"client_ip":     c.IP(),
			"latency":       time.Since(start).String(),
			"request_body":  reqBody,
			"response_body": summarizeBody(c.Response().Body(), c.GetRespHeader(fiber.HeaderContentType)),
		}
		if err != nil {
			fields["error"] = err.Error()
		}

		msg := fmt.Sprintf("%s %s -> %d", c.Method(), c.Path(), status)
		switch {
		case status >= fiber.StatusInternalServerError:
			logs.Error(msg, fields)
		case status >= fiber.StatusBadRequest:
			logs.Warn(msg, fields)
		default:
			logs.Info(msg, fields)
		}
		return err
	}
}

func summarizeRequestBody(c *fiber.Ctx) string {
	if strings.HasPrefix(c.Get(fiber.HeaderContentType), fiber.MIMEMultipartForm) {
		return fmt.Sprintf("<multipart/form-data, %d bytes>", len(c.Body()))
	}
	return summarizeBody(c.Body(), c.Get(fiber.HeaderContentType))
}

func summarizeBody(body []byte, contentType string) string {
	if len(body) == 0 {
		return ""
	}
	if !isTextual(contentType, body) {
		return fmt.Sprintf("<%d bytes>", len(body))
	}
	if len(body) > maxBodyLog {
		return string(body[:maxBodyLog]) + "…(truncated)"
	}
	return string(body)
}

func isTextual(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "application/json") || strings.HasPrefix(ct, "text/") {
		return true
	}

	head := body
	if len(head) > maxBodyLog {
		head = head[:maxBodyLog]
	}
	return utf8.Valid(head)
}
