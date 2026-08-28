package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

const DefaultMaxRequestBodyBytes int64 = 2 * 1024 * 1024

func NewBodyLimitMiddleware(maxBytes int64) echo.MiddlewareFunc {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxRequestBodyBytes
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			if req.ContentLength > maxBytes {
				return WriteError(
					c,
					http.StatusRequestEntityTooLarge,
					"REQUEST_BODY_TOO_LARGE",
					"request body is too large",
				)
			}

			req.Body = http.MaxBytesReader(
				c.Response(),
				req.Body,
				maxBytes,
			)

			return next(c)
		}
	}
}
