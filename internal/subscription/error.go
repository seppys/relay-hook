package subscription

import "errors"

var (
	ErrInvalidID      = errors.New("invalid or missing id")
	ErrInvalidPayload = errors.New("invalid or missing payload")
	ErrInternalServer = errors.New("server error")
)
