package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"sys-design/internal/obs"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start).Seconds()

		obs.APIRequests.WithLabelValues(c.Request.Method, c.FullPath(), intToStatus(c.Writer.Status())).Inc()
		obs.APILatency.WithLabelValues(c.Request.Method, c.FullPath()).Observe(dur)
	}
}

func intToStatus(code int) string {
	return strconv.Itoa(code)
}
