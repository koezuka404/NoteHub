package middleware

import (
	"log"

	"github.com/labstack/echo/v4"
)

func NewRecoveryMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Printf("request_id=%s panic=%v", RequestID(ctx), recovered)
					err = WriteError(ctx, 500, "INTERNAL_ERROR", "内部エラーが発生しました")
				}
			}()
			return next(ctx)
		}
	}
}
