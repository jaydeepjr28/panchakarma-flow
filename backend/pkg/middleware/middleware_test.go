package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCorrelationID(t *testing.T) {
	t.Run("GeneratesNewTraceIDIfMissing", func(t *testing.T) {
		r := gin.New()
		r.Use(middleware.CorrelationID())
		r.GET("/test", func(c *gin.Context) {
			traceID, exists := c.Get(middleware.ContextTraceIDKey)
			if !exists || traceID == "" {
				t.Errorf("expected trace_id in gin.Context")
			}
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		respHeader := w.Header().Get(middleware.HeaderTraceID)
		if respHeader == "" {
			t.Errorf("expected X-Trace-ID response header")
		}
	})

	t.Run("PreservesIncomingTraceID", func(t *testing.T) {
		r := gin.New()
		r.Use(middleware.CorrelationID())
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(middleware.HeaderTraceID, "custom-trace-12345")
		r.ServeHTTP(w, req)

		respHeader := w.Header().Get(middleware.HeaderTraceID)
		if respHeader != "custom-trace-12345" {
			t.Errorf("expected X-Trace-ID 'custom-trace-12345', got '%s'", respHeader)
		}
	})
}

func TestRecovery(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("deliberate test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error, got %d", w.Code)
	}
}
