package middleware

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/koezuka404/notehub/dto"
	"github.com/labstack/echo/v4"
)

const (
	ContextRequestID   = "request_id"
	RequestIDHeader    = "X-Request-ID"
	maxRequestIDLength = 64
)

func NewRequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			requestID := normalizeRequestID(ctx.Request().Header.Get(RequestIDHeader))
			if requestID == "" {
				requestID = uuid.NewString()
			}

			ctx.Set(ContextRequestID, requestID)
			ctx.Response().Header().Set(RequestIDHeader, requestID)
			return next(ctx)
		}
	}
}

func RequestID(ctx echo.Context) string {
	if value, ok := ctx.Get(ContextRequestID).(string); ok {
		return value
	}
	return ""
}

func WriteError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, dto.NewErrorResponse(code, message, RequestID(ctx)))
}

func normalizeRequestID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > maxRequestIDLength {
		return ""
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return ""
	}
	return value
}
