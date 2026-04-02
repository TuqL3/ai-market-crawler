package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RateLimit(rps int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), rps)
	return func(ctx *gin.Context) {
		if !limiter.Allow() {
			ctx.AbortWithStatusPureJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		ctx.Next()
	}
}
