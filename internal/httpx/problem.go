package httpx

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type Problem struct {
	Status  int          `json:"-"`
	Code    ErrorCode    `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

func (p *Problem) Error() string  { return p.Message }
func (p *Problem) GetStatus() int { return p.Status }

func NewProblem(status int, code ErrorCode, msg string, fields ...FieldError) *Problem {
	return &Problem{Status: status, Code: code, Message: msg, Fields: fields}
}

func codeFor(status int) ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return InvalidRequestError
	case http.StatusUnauthorized:
		return UnauthorizedError
	case http.StatusForbidden:
		return ForbiddenError
	case http.StatusUnprocessableEntity:
		return ValidationFailedError
	case http.StatusTooManyRequests:
		return RateLimitedError
	default:
		return InternalError
	}
}

func InstallErrorFormat() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		if status >= 500 {
			log.Printf("server error: %s %v", msg, errs) // never send internals to the client
			return &Problem{Status: status, Code: InternalError, Message: "something went wrong"}
		}
		p := &Problem{Status: status, Code: codeFor(status), Message: msg}
		for _, err := range errs {
			var detail *huma.ErrorDetail
			if errors.As(err, &detail) {
				p.Fields = append(p.Fields, FieldError{
					Field:   strings.TrimPrefix(detail.Location, "body."),
					Message: detail.Message,
				})
			}
		}
		return p
	}
}
