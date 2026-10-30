package errors_test

import (
	"errors"
	"net/http"
	"testing"

	appErrors "github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
)

func TestAppErrorConstructors(t *testing.T) {
	t.Run("BadRequest", func(t *testing.T) {
		cause := errors.New("malformed json")
		err := appErrors.BadRequest("Invalid payload", cause)

		if err.HTTPStatus != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
		}
		if err.Code != appErrors.CodeBadRequest {
			t.Errorf("expected code %s, got %s", appErrors.CodeBadRequest, err.Code)
		}
		if err.Unwrap() != cause {
			t.Errorf("expected unwrap to match cause")
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		details := []appErrors.FieldError{
			{Field: "email", Message: "must be a valid email"},
		}
		err := appErrors.ValidationError("Validation failed", details)

		if err.HTTPStatus != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", err.HTTPStatus)
		}
		if len(err.Details) != 1 {
			t.Fatalf("expected 1 detail, got %d", len(err.Details))
		}
		if err.Details[0].Field != "email" {
			t.Errorf("expected field 'email', got %s", err.Details[0].Field)
		}
	})

	t.Run("Conflict", func(t *testing.T) {
		err := appErrors.Conflict(appErrors.CodeSlotAlreadyHeld, "Slot held by another patient", nil)
		if err.HTTPStatus != http.StatusConflict {
			t.Errorf("expected status 409, got %d", err.HTTPStatus)
		}
		if err.Code != appErrors.CodeSlotAlreadyHeld {
			t.Errorf("expected code %s, got %s", appErrors.CodeSlotAlreadyHeld, err.Code)
		}
	})
}
