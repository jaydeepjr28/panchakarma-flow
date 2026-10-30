package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/health"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthChecker(t *testing.T) {
	t.Run("LiveHandler", func(t *testing.T) {
		checker := health.NewChecker("test-service")
		r := gin.New()
		r.GET("/health/live", checker.LiveHandler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health/live", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}

		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["status"] != "UP" {
			t.Errorf("expected status 'UP', got %s", resp["status"])
		}
	})

	t.Run("ReadyHandler_AllHealthy", func(t *testing.T) {
		checker := health.NewChecker("test-service")
		checker.RegisterCheck("database", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("redis", func(ctx context.Context) error {
			return nil
		})

		r := gin.New()
		r.GET("/health/ready", checker.ReadyHandler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}

		var report health.Report
		_ = json.Unmarshal(w.Body.Bytes(), &report)
		if report.Status != "UP" {
			t.Errorf("expected status 'UP', got %s", report.Status)
		}
		if report.Components["database"].Status != "UP" {
			t.Errorf("expected database 'UP', got %s", report.Components["database"].Status)
		}
	})

	t.Run("ReadyHandler_ComponentDegraded", func(t *testing.T) {
		checker := health.NewChecker("test-service")
		checker.RegisterCheck("database", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("redis", func(ctx context.Context) error {
			return errors.New("connection refused")
		})

		r := gin.New()
		r.GET("/health/ready", checker.ReadyHandler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
		}

		var report health.Report
		_ = json.Unmarshal(w.Body.Bytes(), &report)
		if report.Status != "DEGRADED" {
			t.Errorf("expected status 'DEGRADED', got %s", report.Status)
		}
		if report.Components["redis"].Status != "DOWN" {
			t.Errorf("expected redis 'DOWN', got %s", report.Components["redis"].Status)
		}
	})
}
