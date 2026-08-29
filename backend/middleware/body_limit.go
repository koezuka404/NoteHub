package middleware

import (
	"bytes"
	"errors"
	"io"
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

			if !methodMayHaveBody(req.Method) {
				return next(c)
			}

			if req.ContentLength > maxBytes {
				return writeBodyTooLarge(c)
			}

			if req.Body == nil {
				return next(c)
			}

			limited := http.MaxBytesReader(c.Response(), req.Body, maxBytes)
			body, err := io.ReadAll(limited)
			_ = limited.Close()
			if err != nil {
				if isRequestBodyTooLarge(err) {
					return writeBodyTooLarge(c)
				}
				return err
			}

			req.Body = io.NopCloser(bytes.NewReader(body))
			req.ContentLength = int64(len(body))
			return next(c)
		}
	}
}

func methodMayHaveBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	return errors.Is(err, echo.ErrStatusRequestEntityTooLarge)
}

func writeBodyTooLarge(c echo.Context) error {
	return WriteError(
		c,
		http.StatusRequestEntityTooLarge,
		"REQUEST_BODY_TOO_LARGE",
		"リクエストが大きすぎます",
	)
}
