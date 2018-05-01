package clairclient

import "fmt"

type statusCodeErrorInt interface {
	statusCode() int
}

type statusCodeError struct {
	code int
	error
}

func (e statusCodeError) statusCode() int {
	return e.code
}

func (e statusCodeError) Error() string {
	return fmt.Sprintf("recieved unexpected response status code %d", e.code)
}

func newStatusCodeError(code int) statusCodeError {
	return statusCodeError{
		code: code,
	}
}

// IsStatusCodeError func init
func IsStatusCodeError(err error) bool {
	_, ok := err.(statusCodeErrorInt)
	return ok
}

// ErrorStatusCode func init
func ErrorStatusCode(err error) int {
	e, ok := err.(statusCodeError)
	if ok {
		return e.statusCode()
	}
	return -1
}
