package errors

import (
	"fmt"
	"net/http"
	"time"
)

// Standard Domain Error Codes
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeValidationError     = "VALIDATION_ERROR"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeInternalError       = "INTERNAL_SERVER_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	CodeSlotAlreadyHeld     = "SLOT_ALREADY_HELD"
	CodeDoctorOnLeave       = "DOCTOR_ON_LEAVE"
	CodeCapacityExceeded    = "CAPACITY_EXCEEDED"
	CodeDoubleBooking       = "DOUBLE_BOOKING_PREVENTED"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeTokenRevoked        = "TOKEN_REVOKED"
	CodePreChecklistFailed  = "PRE_CHECKLIST_FAILED"
	CodeComplicationFlagged = "COMPLICATION_FLAGGED"
)

// AppError represents a structured, client-safe error response.
type AppError struct {
	HTTPStatus int           `json:"-"`
	Code       string        `json:"code"`
	Message    string        `json:"message"`
	TraceID    string        `json:"trace_id,omitempty"`
	Timestamp  string        `json:"timestamp"`
	Details    []FieldError  `json:"details,omitempty"`
	Err        error         `json:"-"` // Underlying cause for internal logging
}

// FieldError represents a field-level validation issue.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError with timestamp.
func New(httpStatus int, code, message string, underlying error) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Err:        underlying,
	}
}

// Convenience Constructors
func BadRequest(message string, err error) *AppError {
	return New(http.StatusBadRequest, CodeBadRequest, message, err)
}

func ValidationError(message string, details []FieldError) *AppError {
	appErr := New(http.StatusUnprocessableEntity, CodeValidationError, message, nil)
	appErr.Details = details
	return appErr
}

func Unauthorized(message string, err error) *AppError {
	return New(http.StatusUnauthorized, CodeUnauthorized, message, err)
}

func Forbidden(message string, err error) *AppError {
	return New(http.StatusForbidden, CodeForbidden, message, err)
}

func NotFound(message string, err error) *AppError {
	return New(http.StatusNotFound, CodeNotFound, message, err)
}

func Conflict(code, message string, err error) *AppError {
	if code == "" {
		code = CodeConflict
	}
	return New(http.StatusConflict, code, message, err)
}

func Internal(message string, err error) *AppError {
	return New(http.StatusInternalServerError, CodeInternalError, message, err)
}

func ServiceUnavailable(message string, err error) *AppError {
	return New(http.StatusServiceUnavailable, CodeServiceUnavailable, message, err)
}
