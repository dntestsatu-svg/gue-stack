package apperror

import "fmt"

type AppError struct {
	StatusCode int
	Message    string
	Details    any
}

func (e *AppError) Error() string {
	if e.Details == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Details)
}

func New(status int, message string, details any) *AppError {
	return &AppError{StatusCode: status, Message: message, Details: details}
}
