package validation

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// DatabaseError represents common database errors
type DatabaseError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Error implements the error interface
func (de *DatabaseError) Error() string {
	return de.Message
}

// Common database error types
const (
	ErrorTypeUniqueViolation    = "unique_violation"
	ErrorTypeForeignKeyViolation = "foreign_key_violation"
	ErrorTypeNotFound           = "not_found"
	ErrorTypeInternal           = "internal"
)

// ParseDatabaseError converts database errors into user-friendly errors
func ParseDatabaseError(err error) error {
	if err == nil {
		return nil
	}

	// Check for PostgreSQL error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return handleUniqueViolation(pgErr)
		case "23503": // foreign_key_violation
			return handleForeignKeyViolation(pgErr)
		case "23502": // not_null_violation
			return handleNotNullViolation(pgErr)
		case "23514": // check_violation
			return handleCheckViolation(pgErr)
		}
	}

	// Return original error if not a known database error
	return err
}

// handleUniqueViolation processes unique constraint violations
func handleUniqueViolation(pgErr *pgconn.PgError) error {
	// Parse the constraint name to determine the field
	field := parseFieldFromConstraint(pgErr.ConstraintName)

	var message string
	switch {
	case strings.Contains(pgErr.ConstraintName, "url"):
		message = "A repository with this URL already exists"
	case strings.Contains(pgErr.ConstraintName, "email"):
		message = "This email is already in use"
	default:
		message = "This value already exists in the system"
	}

	return &DatabaseError{
		Type:    ErrorTypeUniqueViolation,
		Message: message,
		Field:   field,
	}
}

// handleForeignKeyViolation processes foreign key violations
func handleForeignKeyViolation(pgErr *pgconn.PgError) error {
	var message string
	if strings.Contains(pgErr.Message, "is still referenced") {
		message = "Cannot delete this record because it is referenced by other records"
	} else {
		message = "Referenced record does not exist"
	}

	return &DatabaseError{
		Type:    ErrorTypeForeignKeyViolation,
		Message: message,
	}
}

// handleNotNullViolation processes not null constraint violations
func handleNotNullViolation(pgErr *pgconn.PgError) error {
	field := pgErr.ColumnName
	if field == "" {
		field = parseFieldFromConstraint(pgErr.ConstraintName)
	}

	return &DatabaseError{
		Type:    ErrorTypeInternal,
		Message: "Required field is missing: " + field,
		Field:   field,
	}
}

// handleCheckViolation processes check constraint violations
func handleCheckViolation(pgErr *pgconn.PgError) error {
	return &DatabaseError{
		Type:    ErrorTypeInternal,
		Message: "Invalid value for field",
	}
}

// parseFieldFromConstraint extracts the field name from a constraint name
// Example: "repositories_url_key" -> "url"
func parseFieldFromConstraint(constraintName string) string {
	if constraintName == "" {
		return ""
	}

	parts := strings.Split(constraintName, "_")
	if len(parts) >= 2 {
		// Return the second-to-last part (field name)
		// Example: repositories_url_key -> url
		return parts[len(parts)-2]
	}
	return constraintName
}

// IsUniqueViolation checks if an error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return dbErr.Type == ErrorTypeUniqueViolation
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found")
}
