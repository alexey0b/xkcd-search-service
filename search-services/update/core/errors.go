package core

import "errors"

var (
	ErrBadArguments       = errors.New("arguments are not acceptable")
	ErrAlreadyExists      = errors.New("resource or task already exists")
	ErrNotFound           = errors.New("resource is not found")
	ErrServiceUnavailable = errors.New("service is currently unavailable")
)
