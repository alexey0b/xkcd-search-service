package core

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrBadArguments       = errors.New("arguments are not acceptable")
	ErrAlreadyExists      = errors.New("resource or task already exists")
	ErrServiceUnavailable = errors.New("service is currently unavailable")
)
