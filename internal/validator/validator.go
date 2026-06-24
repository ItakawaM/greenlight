package validator

import (
	"regexp"
	"slices"
)

// EmailRX is a standardized regular expression for emails.
var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// Validator is responsible for recording error messages.
type Validator struct {
	Errors map[string]string
}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

// Valid checks if no errors were encountered.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError records an error if it does not exist already.
func (v *Validator) AddError(key string, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check records an error if the provided boolean expression is not okay.
func (v *Validator) Check(ok bool, key string, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// PermittedValue checks if the provided value is allowed.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

// Matches checks if the provided value matches the regular expression.
func Matches(value string, expression *regexp.Regexp) bool {
	return expression.MatchString(value)
}

// Unique checks if all of the provided values are unique.
func Unique[T comparable](values []T) bool {
	unique := make(map[T]bool)

	for _, value := range values {
		if unique[value] {
			return false
		}
		unique[value] = true
	}

	return len(unique) == len(values)
}
