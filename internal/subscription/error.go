package subscription

import "errors"

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrNotAllowed     = errors.New("not allowed")
	ErrInvalidID      = errors.New("invalid or missing id")
	ErrInvalidPayload = errors.New("invalid or missing payload")
	ErrInternalServer = errors.New("server error")
)
