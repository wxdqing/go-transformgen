package frame

import "errors"

var (
	ErrShortFrame      = errors.New("transformgen/frame: short frame")
	ErrBodyLenMismatch = errors.New("transformgen/frame: body length mismatch")
)
