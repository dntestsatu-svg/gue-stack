package repository

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
