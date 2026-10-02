package httpx

type ErrorCode string

const (
	InternalError           ErrorCode = "InternalError"
	InvalidRequestError     ErrorCode = "InvalidRequestError"
	ValidationFailedError   ErrorCode = "ValidationFailedError"
	UsernameTakenError      ErrorCode = "UsernameTakenError"
	EmailAlreadyInUseError  ErrorCode = "EmailAlreadyInUseError"
	InvalidCredentialsError ErrorCode = "InvalidCredentialsError"
	UnauthorizedError       ErrorCode = "UnauthorizedError"
)
