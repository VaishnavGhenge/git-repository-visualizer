package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	if len(ve.Errors) == 1 {
		return ve.Errors[0].Message
	}
	return fmt.Sprintf("validation failed with %d errors", len(ve.Errors))
}

// HasErrors returns true if there are validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Add adds a validation error
func (ve *ValidationErrors) Add(field, message string) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// Validator provides common validation methods
type Validator struct {
	errors ValidationErrors
}

// New creates a new Validator instance
func New() *Validator {
	return &Validator{
		errors: ValidationErrors{Errors: []ValidationError{}},
	}
}

// Required validates that a field is not empty
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors.Add(field, fmt.Sprintf("%s is required", field))
	}
	return v
}

// MinLength validates minimum string length
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.errors.Add(field, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
	return v
}

// MaxLength validates maximum string length
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.errors.Add(field, fmt.Sprintf("%s must not exceed %d characters", field, max))
	}
	return v
}

// URL validates that a string is a valid URL
func (v *Validator) URL(field, value string) *Validator {
	if value == "" {
		return v
	}

	parsedURL, err := url.Parse(value)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		v.errors.Add(field, fmt.Sprintf("%s must be a valid URL", field))
	}
	return v
}

// GitURL validates that a string is a valid Git repository URL
func (v *Validator) GitURL(field, value string) *Validator {
	if value == "" {
		return v
	}

	// Support HTTP(S) and SSH Git URLs
	httpPattern := regexp.MustCompile(`^https?://[^/]+/.+\.git$|^https?://[^/]+/.+$`)
	sshPattern := regexp.MustCompile(`^git@[^:]+:.+\.git$|^ssh://git@[^/]+/.+$`)

	if !httpPattern.MatchString(value) && !sshPattern.MatchString(value) {
		v.errors.Add(field, fmt.Sprintf("%s must be a valid Git repository URL (HTTP(S) or SSH)", field))
	}
	return v
}

// Email validates that a string is a valid email address
func (v *Validator) Email(field, value string) *Validator {
	if value == "" {
		return v
	}

	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(value) {
		v.errors.Add(field, fmt.Sprintf("%s must be a valid email address", field))
	}
	return v
}

// InRange validates that an integer is within a range
func (v *Validator) InRange(field string, value, min, max int) *Validator {
	if value < min || value > max {
		v.errors.Add(field, fmt.Sprintf("%s must be between %d and %d", field, min, max))
	}
	return v
}

// GreaterThan validates that an integer is greater than a minimum
func (v *Validator) GreaterThan(field string, value, min int) *Validator {
	if value <= min {
		v.errors.Add(field, fmt.Sprintf("%s must be greater than %d", field, min))
	}
	return v
}

// GreaterThanOrEqual validates that an integer is greater than or equal to a minimum
func (v *Validator) GreaterThanOrEqual(field string, value, min int) *Validator {
	if value < min {
		v.errors.Add(field, fmt.Sprintf("%s must be greater than or equal to %d", field, min))
	}
	return v
}

// OneOf validates that a value is one of the allowed values
func (v *Validator) OneOf(field, value string, allowed []string) *Validator {
	if value == "" {
		return v
	}

	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.errors.Add(field, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
	return v
}

// Matches validates that a string matches a regex pattern
func (v *Validator) Matches(field, value, pattern, message string) *Validator {
	if value == "" {
		return v
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil || !matched {
		if message == "" {
			message = fmt.Sprintf("%s format is invalid", field)
		}
		v.errors.Add(field, message)
	}
	return v
}

// Custom allows for custom validation logic
func (v *Validator) Custom(field string, fn func() error) *Validator {
	if err := fn(); err != nil {
		v.errors.Add(field, err.Error())
	}
	return v
}

// Validate returns the validation errors if any exist
func (v *Validator) Validate() error {
	if v.errors.HasErrors() {
		return &v.errors
	}
	return nil
}

// Errors returns the validation errors
func (v *Validator) Errors() *ValidationErrors {
	return &v.errors
}
