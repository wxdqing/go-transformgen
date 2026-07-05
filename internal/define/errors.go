package define

import "errors"

var (
	ErrUnsupportedVersion = errors.New("transformgen/define: unsupported version")
	ErrInvalidModuleName  = errors.New("transformgen/define: invalid module name")
)
