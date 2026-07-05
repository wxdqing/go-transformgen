package registry

import "errors"

var (
	ErrDuplicateMessageID  = errors.New("transformgen/registry: duplicate message id")
	ErrUnknownMessageID    = errors.New("transformgen/registry: unknown message id")
	ErrMessageKindMismatch = errors.New("transformgen/registry: message kind mismatch")
	ErrDuplicateHandler    = errors.New("transformgen/registry: duplicate handler")
	ErrHandlerNotFound     = errors.New("transformgen/registry: handler not found")
	ErrInvalidContextType  = errors.New("transformgen/registry: invalid context type")
	ErrInvalidMessageType  = errors.New("transformgen/registry: invalid message type")
)
