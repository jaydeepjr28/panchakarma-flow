package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

type RouteMap struct {
	IdentityURL  *url.URL
	ClinicURL    *url.URL
	ApptURL      *url.URL
	TherapyURL   *url.URL
	NotifURL     *url.URL
	FeedbackURL  *url.URL
}

func NewReverseProxy(target *url.URL) gin.HandlerFunc {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Preserve incoming X-Trace-ID
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		if traceID := req.Header.Get(middleware.HeaderTraceID); traceID != "" {
			req.Header.Set(middleware.HeaderTraceID, traceID)
		}
	}

	// Strip duplicate downstream CORS headers so only the API Gateway's CORS middleware applies
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Expose-Headers")
		resp.Header.Del("Access-Control-Max-Age")
		return nil
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func RegisterGatewayRoutes(router *gin.Engine, routes RouteMap) {
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	api := router.Group("/api/v1")
	{
		// Identity routes
		if routes.IdentityURL != nil {
			proxy := NewReverseProxy(routes.IdentityURL)
			api.Any("/auth", proxy)
			api.Any("/auth/*action", proxy)
			api.Any("/patients", proxy)
			api.Any("/patients/*action", proxy)
		}

		// Clinic routes
		if routes.ClinicURL != nil {
			proxy := NewReverseProxy(routes.ClinicURL)
			api.Any("/clinics", proxy)
			api.Any("/clinics/*action", proxy)
			api.Any("/doctors", proxy)
			api.Any("/doctors/*action", proxy)
			api.Any("/packages", proxy)
			api.Any("/packages/*action", proxy)
		}

		// Appointment routes
		if routes.ApptURL != nil {
			proxy := NewReverseProxy(routes.ApptURL)
			api.Any("/appointments", proxy)
			api.Any("/appointments/*action", proxy)
			api.Any("/bookings", proxy)
			api.Any("/bookings/*action", proxy)
			api.Any("/availability", proxy)
			api.Any("/availability/*action", proxy)
		}

		// Therapy routes
		if routes.TherapyURL != nil {
			proxy := NewReverseProxy(routes.TherapyURL)
			api.Any("/sessions", proxy)
			api.Any("/sessions/*action", proxy)
		}

		// Notification routes
		if routes.NotifURL != nil {
			proxy := NewReverseProxy(routes.NotifURL)
			api.Any("/notifications", proxy)
			api.Any("/notifications/*action", proxy)
		}

		// Feedback & Analytics routes
		if routes.FeedbackURL != nil {
			proxy := NewReverseProxy(routes.FeedbackURL)
			api.Any("/feedback", proxy)
			api.Any("/feedback/*action", proxy)
			api.Any("/analytics", proxy)
			api.Any("/analytics/*action", proxy)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			response.Error(c, errors.NotFound("Route not found on API Gateway", nil))
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})
}
