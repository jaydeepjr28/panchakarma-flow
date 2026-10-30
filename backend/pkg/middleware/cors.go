package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS configures cross-origin resource sharing for frontend clients.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		isAllowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Trace-ID, X-Correlation-ID, X-Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
			c.Header("Access-Control-Expose-Headers", "X-Trace-ID, X-Correlation-ID")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
