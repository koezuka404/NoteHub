package middleware

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func NewLoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			start := time.Now()
			err := next(ctx)
			status := ctx.Response().Status
			if err != nil {
				if httpErr, ok := err.(*echo.HTTPError); ok {
					status = httpErr.Code
				}
			}
			if status == 0 {
				status = 200
			}

			userID := "-"
			if id, ok := ctx.Get(ContextUserID).(uuid.UUID); ok && id != uuid.Nil {
				userID = id.String()
			}

			log.Printf(
				"request_id=%s method=%s path=%s status=%d duration=%s user_id=%s",
				RequestID(ctx),
				ctx.Request().Method,
				ctx.Path(),
				status,
				time.Since(start).Round(time.Millisecond),
				userID,
			)
			return err
		}
	}
}
