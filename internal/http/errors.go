package http

import (
	"errors"
	"net/http"

	"git-repository-visualizer/internal/validation"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string                     `json:"error"`
	Details interface{}                `json:"details,omitempty"`
	Code    string                     `json:"code,omitempty"`
}

// Error writes an error response to the client
func Error(w http.ResponseWriter, err error, statusCode int) {
	// Parse validation errors
	var validationErr *validation.ValidationErrors
	if errors.As(err, &validationErr) {
		JSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: validationErr.Errors,
			Code:    "VALIDATION_ERROR",
		})
		return
	}

	// Parse database errors
	var dbErr *validation.DatabaseError
	if errors.As(err, &dbErr) {
		// Map database errors to appropriate HTTP status codes
		status := mapDatabaseErrorToHTTPStatus(dbErr)
		JSON(w, status, ErrorResponse{
			Error:   dbErr.Message,
			Code:    dbErr.Type,
			Details: map[string]string{"field": dbErr.Field},
		})
		return
	}

	// Default error response
	JSON(w, statusCode, ErrorResponse{
		Error: err.Error(),
	})
}

// mapDatabaseErrorToHTTPStatus maps database error types to HTTP status codes
func mapDatabaseErrorToHTTPStatus(dbErr *validation.DatabaseError) int {
	switch dbErr.Type {
	case validation.ErrorTypeUniqueViolation:
		return http.StatusConflict
	case validation.ErrorTypeForeignKeyViolation:
		return http.StatusBadRequest
	case validation.ErrorTypeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
