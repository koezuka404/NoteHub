package middleware

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type ITokenBucketLimiter interface {
	Allow(ctx context.Context, key string, capacity int, refillPerSecond float64, now time.Time) (bool, time.Duration, error)
}

type RateLimitConfig struct {
	Capacity        int
	RefillPerSecond float64
}

type RateLimitMiddleware struct {
	limiter ITokenBucketLimiter
	config  RateLimitConfig
	now     func() time.Time
}

func NewRateLimitMiddleware(limiter ITokenBucketLimiter, config RateLimitConfig) echo.MiddlewareFunc {
	middleware := &RateLimitMiddleware{limiter: limiter, config: config, now: time.Now}
	return middleware.Handle
}

func (m *RateLimitMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		key := ctx.RealIP() + ":" + ctx.Request().Method + ":" + ctx.Path()
		allowed, retryAfter, err := m.limiter.Allow(
			ctx.Request().Context(), key,
			m.config.Capacity, m.config.RefillPerSecond, m.now().UTC(),
		)
		if err != nil {
			return WriteError(ctx, http.StatusServiceUnavailable, "RATE_LIMIT_SERVICE_UNAVAILABLE", "アクセス制限サービスを利用できません")
		}
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			ctx.Response().Header().Set("Retry-After", strconv.Itoa(seconds))
			return WriteError(ctx, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "リクエスト回数が上限を超えました")
		}
		return next(ctx)
	}
}
