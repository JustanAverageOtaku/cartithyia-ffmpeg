package feature

import "errors"

type (
	Feature interface {
		Execute(args []string) error
	}
)

var (
	ErrEmptySource      = errors.New("source cannot be empty")
	ErrEmptyDestination = errors.New("destination cannot be empty")
	ErrNotFile          = errors.New("source cannot be a directory")
)
