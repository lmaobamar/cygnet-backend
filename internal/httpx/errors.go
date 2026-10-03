package httpx

import (
	"errors"
	"log"
	"net/http"
)

type ErrorCode string

const (
	InternalError           ErrorCode = "InternalError"
	InvalidRequestError     ErrorCode = "InvalidRequestError"
	ValidationFailedError   ErrorCode = "ValidationFailedError"
	HandleTakenError        ErrorCode = "HandleTakenError"
	EmailAlreadyInUseError  ErrorCode = "EmailAlreadyInUseError"
	InvalidCredentialsError ErrorCode = "InvalidCredentialsError"
	UnauthorizedError       ErrorCode = "UnauthorizedError"
	ForbiddenError          ErrorCode = "ForbiddenError"
	RateLimitedError        ErrorCode = "RateLimitedError"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code    ErrorCode    `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type APIError struct {
	Status int
	Body   ErrorResponse
}

func (e *APIError) Error() string { return e.Body.Message }

func NewError(status int, code ErrorCode, msg string, fields ...FieldError) *APIError {
	return &APIError{Status: status, Body: ErrorResponse{Code: code, Message: msg, Fields: fields}}
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func Handle(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err == nil {
			return
		}
		if apiErr, ok := errors.AsType[*APIError](err); ok {
			WriteJson(w, apiErr.Status, apiErr.Body)
			return
		}
		log.Printf("unhandled error on %s %s: %v", r.Method, r.URL.Path, err)
		WriteJson(w, http.StatusInternalServerError, ErrorResponse{
			Code: InternalError, Message: "something went wrong",
		})
	}
}
