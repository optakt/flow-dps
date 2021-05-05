package model

import (
	"errors"
)

var (
	ErrFinished = errors.New("finished")
	ErrTimeout  = errors.New("timeout")
)
