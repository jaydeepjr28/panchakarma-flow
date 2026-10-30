package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

const (
	HeaderTraceID       = "X-Trace-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	ContextTraceIDKey   = "trace_id"
)

// CorrelationID ensures every incoming request has an immutable trace/correlation ID.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(HeaderTraceID)
		if traceID == "" {
			traceID = c.GetHeader(HeaderCorrelationID)
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Set(ContextTraceIDKey, traceID)
		c.Header(HeaderTraceID, traceID)

		// Inject into standard request context for downstream services
		ctx := logger.ContextWithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
