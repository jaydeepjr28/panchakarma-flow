package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// RequestLogger logs incoming HTTP requests with latency, method, path, status, and trace_id.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		traceID, _ := c.Get(ContextTraceIDKey)
		traceIDStr, _ := traceID.(string)

		logLevel := slog.LevelInfo
		if status >= 500 {
			logLevel = slog.LevelError
		} else if status >= 400 {
			logLevel = slog.LevelWarn
		}

		logger.WithContext(c.Request.Context()).Log(
			c.Request.Context(),
			logLevel,
			"HTTP Request",
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("client_ip", clientIP),
			slog.String("trace_id", traceIDStr),
		)
	}
}
