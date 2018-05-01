// Package commonerr defines reusable error types common throughout the Port
// Authority codebase.
package commonerr

import "errors"

var (
	// ErrNotFound occurs when a resource could not be found
	ErrNotFound = errors.New("the resource cannot be found")
)

// ErrBadRequest occurs when a method has been passed an inappropriate argument
type ErrBadRequest struct {
	s string
}

// NewBadRequestError instantiates a ErrBadRequest with the specified message
func NewBadRequestError(message string) error {
	return &ErrBadRequest{s: message}
}

func (e *ErrBadRequest) Error() string {
	return e.s
}
