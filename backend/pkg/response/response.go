package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
)

// Response represents the standard API response envelope.
type Response struct {
	Success bool             `json:"success"`
	Data    any              `json:"data,omitempty"`
	Error   *errors.AppError `json:"error,omitempty"`
}

// Success writes a successful JSON response with HTTP 200/201.
func Success(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
	})
}

// OK writes an HTTP 200 OK JSON response.
func OK(c *gin.Context, data any) {
	Success(c, http.StatusOK, data)
}

// Created writes an HTTP 201 Created JSON response.
func Created(c *gin.Context, data any) {
	Success(c, http.StatusCreated, data)
}

// Error writes a structured error JSON response.
func Error(c *gin.Context, appErr *errors.AppError) {
	if appErr == nil {
		appErr = errors.Internal("An unexpected error occurred", nil)
	}

	// Attach trace ID from context if present
	if traceID, exists := c.Get("trace_id"); exists {
		if tidStr, ok := traceID.(string); ok && appErr.TraceID == "" {
			appErr.TraceID = tidStr
		}
	}

	c.JSON(appErr.HTTPStatus, Response{
		Success: false,
		Error:   appErr,
	})
}
