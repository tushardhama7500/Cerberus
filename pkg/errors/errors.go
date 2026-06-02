package errors

import "fmt"

// AppError is our domain error type.
// We wrap these before sending to GraphQL so we control what the client sees.
// Never expose raw DB errors or stack traces to clients.
type AppError struct {
	Code    string
	Message string
	Err     error // Underlying error for internal logging — not sent to client
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

// Named constructors for each error category

func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg}
}

func NotFound(resource string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: fmt.Sprintf("%s not found", resource)}
}

func ValidationError(msg string) *AppError {
	return &AppError{Code: "VALIDATION_ERROR", Message: msg}
}

func Internal(msg string, err error) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: msg, Err: err}
}

func Conflict(msg string) *AppError {
	return &AppError{Code: "CONFLICT", Message: msg}
}
