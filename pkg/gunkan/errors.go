package gunkan

import (
	"errors"
)

var (
	ErrNotFound      = errors.New("404/Not-Found")
	ErrForbidden     = errors.New("403/Forbidden")
	ErrAlreadyExists = errors.New("409/Conflict")
	ErrStorageError  = errors.New("502/Backend-Error")
	ErrInternalError = errors.New("500/Internal Error")
)

func MapCodeToError(code int) error {
	switch code {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 409:
		return ErrAlreadyExists
	case 200, 201, 204:
		return nil
	default:
		return ErrInternalError
	}
}
