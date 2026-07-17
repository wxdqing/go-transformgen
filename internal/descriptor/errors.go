package descriptor

import "errors"

var (
	ErrDuplicateMessage = errors.New("transformgen/descriptor: duplicate message")
	ErrUnsupportedField = errors.New("transformgen/descriptor: unsupported field")
)
