package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/render"
)

func TestErrInvalidFields(t *testing.T) {
	if ErrInvalidFields == nil {
		t.Error("ErrInvalidFields should not be nil")
	}

	expected := "invalid fields in request"
	if ErrInvalidFields.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, ErrInvalidFields.Error())
	}
}

func TestValidationError(t *testing.T) {
	validationErr := ValidationError{
		Path:    "field.name",
		Message: "field is required",
	}

	if validationErr.Path != "field.name" {
		t.Errorf("Expected Path 'field.name', got '%s'", validationErr.Path)
	}

	if validationErr.Message != "field is required" {
		t.Errorf("Expected Message 'field is required', got '%s'", validationErr.Message)
	}
}

func TestErrResponse_Render(t *testing.T) {
	tests := []struct {
		name           string
		errResponse    *ErrResponse
		expectedStatus int
	}{
		{
			name: "Bad Request",
			errResponse: &ErrResponse{
				HTTPStatusCode: http.StatusBadRequest,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Not Found",
			errResponse: &ErrResponse{
				HTTPStatusCode: http.StatusNotFound,
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Internal Server Error",
			errResponse: &ErrResponse{
				HTTPStatusCode: http.StatusInternalServerError,
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			err := tt.errResponse.Render(w, r)
			if err != nil {
				t.Errorf("Render() returned error: %v", err)
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestErrInvalidRequest(t *testing.T) {
	testErr := errors.New("test validation error")

	renderer := ErrInvalidRequest(testErr)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.Err != testErr {
		t.Errorf("Expected Err to be %v, got %v", testErr, errResp.Err)
	}

	if errResp.ErrorText != testErr.Error() {
		t.Errorf("Expected ErrorText '%s', got '%s'", testErr.Error(), errResp.ErrorText)
	}

	if errResp.HTTPStatusCode != http.StatusBadRequest {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusBadRequest, errResp.HTTPStatusCode)
	}

	if errResp.StatusText != "Invalid request" {
		t.Errorf("Expected StatusText 'Invalid request', got '%s'", errResp.StatusText)
	}

	if errResp.ValidationErrors != nil {
		t.Error("Expected ValidationErrors to be nil")
	}
}

func TestMultiErrInvalidRequest(t *testing.T) {
	validationErrs := []ValidationError{
		{Path: "name", Message: "name is required"},
		{Path: "email", Message: "email is invalid"},
	}

	renderer := MultiErrInvalidRequest(validationErrs)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.Err != ErrInvalidFields {
		t.Errorf("Expected Err to be %v, got %v", ErrInvalidFields, errResp.Err)
	}

	if errResp.ErrorText != ErrInvalidFields.Error() {
		t.Errorf("Expected ErrorText '%s', got '%s'", ErrInvalidFields.Error(), errResp.ErrorText)
	}

	if errResp.HTTPStatusCode != http.StatusBadRequest {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusBadRequest, errResp.HTTPStatusCode)
	}

	if errResp.StatusText != "Invalid request" {
		t.Errorf("Expected StatusText 'Invalid request', got '%s'", errResp.StatusText)
	}

	if len(errResp.ValidationErrors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(errResp.ValidationErrors))
	}

	if errResp.ValidationErrors[0].Path != "name" {
		t.Errorf("Expected first validation error path 'name', got '%s'", errResp.ValidationErrors[0].Path)
	}

	if errResp.ValidationErrors[1].Message != "email is invalid" {
		t.Errorf("Expected second validation error message 'email is invalid', got '%s'", errResp.ValidationErrors[1].Message)
	}
}

func TestErrNotFound(t *testing.T) {
	testErr := errors.New("resource not found")

	renderer := ErrNotFound(testErr)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.Err != testErr {
		t.Errorf("Expected Err to be %v, got %v", testErr, errResp.Err)
	}

	if errResp.ErrorText != testErr.Error() {
		t.Errorf("Expected ErrorText '%s', got '%s'", testErr.Error(), errResp.ErrorText)
	}

	if errResp.HTTPStatusCode != http.StatusNotFound {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusNotFound, errResp.HTTPStatusCode)
	}

	if errResp.StatusText != "Resource not found" {
		t.Errorf("Expected StatusText 'Resource not found', got '%s'", errResp.StatusText)
	}
}

func TestErrInternal(t *testing.T) {
	testErr := errors.New("database connection failed")

	renderer := ErrInternal(testErr)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.Err != testErr {
		t.Errorf("Expected Err to be %v, got %v", testErr, errResp.Err)
	}

	if errResp.ErrorText != testErr.Error() {
		t.Errorf("Expected ErrorText '%s', got '%s'", testErr.Error(), errResp.ErrorText)
	}

	if errResp.HTTPStatusCode != http.StatusInternalServerError {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusInternalServerError, errResp.HTTPStatusCode)
	}

	if errResp.StatusText != "Internal server error" {
		t.Errorf("Expected StatusText 'Internal server error', got '%s'", errResp.StatusText)
	}
}

func TestErrUnauthenticated(t *testing.T) {
	testErr := errors.New("invalid credentials")

	renderer := ErrUnauthenticated(testErr)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.Err != testErr {
		t.Errorf("Expected Err to be %v, got %v", testErr, errResp.Err)
	}

	if errResp.ErrorText != testErr.Error() {
		t.Errorf("Expected ErrorText '%s', got '%s'", testErr.Error(), errResp.ErrorText)
	}

	if errResp.HTTPStatusCode != http.StatusUnauthorized {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusUnauthorized, errResp.HTTPStatusCode)
	}

	if errResp.StatusText != "Unauthorized" {
		t.Errorf("Expected StatusText 'Unauthorized', got '%s'", errResp.StatusText)
	}
}

func TestErrUnauthorized(t *testing.T) {
	testErr := errors.New("insufficient permissions")

	renderer := ErrUnauthorized(testErr)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.Err != testErr {
		t.Errorf("Expected Err to be %v, got %v", testErr, errResp.Err)
	}

	if errResp.ErrorText != testErr.Error() {
		t.Errorf("Expected ErrorText '%s', got '%s'", testErr.Error(), errResp.ErrorText)
	}

	if errResp.HTTPStatusCode != http.StatusForbidden {
		t.Errorf("Expected HTTPStatusCode %d, got %d", http.StatusForbidden, errResp.HTTPStatusCode)
	}

	if errResp.StatusText != "Forbidden" {
		t.Errorf("Expected StatusText 'Forbidden', got '%s'", errResp.StatusText)
	}
}

func TestErrResponse_ImplementsRenderer(t *testing.T) {
	var _ render.Renderer = &ErrResponse{}
}

func TestMultiErrInvalidRequest_EmptyValidationErrors(t *testing.T) {
	var validationErrs []ValidationError

	renderer := MultiErrInvalidRequest(validationErrs)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if len(errResp.ValidationErrors) != 0 {
		t.Errorf("Expected 0 validation errors, got %d", len(errResp.ValidationErrors))
	}
}

func TestMultiErrInvalidRequest_NilValidationErrors(t *testing.T) {
	renderer := MultiErrInvalidRequest(nil)
	errResp, ok := renderer.(*ErrResponse)
	if !ok {
		t.Fatal("Expected *ErrResponse type")
	}

	if errResp.ValidationErrors != nil {
		t.Error("Expected ValidationErrors to be nil")
	}
}
