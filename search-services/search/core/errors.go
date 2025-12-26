package core

import "errors"

var (
	ErrBadArguments       = errors.New("arguments are not acceptable")
	ErrServiceUnavailable = errors.New("service is currently unavailable")
)
