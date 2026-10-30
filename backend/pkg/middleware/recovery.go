package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

// Recovery recovers from panics and writes a clean JSON error response without leaking internals.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				errStr := fmt.Sprintf("%v", r)

				logger.WithContext(c.Request.Context()).Error(
					"Panic recovered in HTTP handler",
					slog.String("error", errStr),
					slog.String("stack", stack),
				)

				appErr := errors.Internal("Internal server error", nil)
				response.Error(c, appErr)
				c.Abort()
			}
		}()
		c.Next()
	}
}
