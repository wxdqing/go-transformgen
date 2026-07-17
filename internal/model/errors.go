package model

import "errors"

var (
	ErrMessageNotFound      = errors.New("transformgen/model: message not found")
	ErrMessageKindMismatch  = errors.New("transformgen/model: message kind mismatch")
	ErrDuplicateRequest     = errors.New("transformgen/model: duplicate request binding")
	ErrDuplicateMessageName = errors.New("transformgen/model: duplicate message short name")
	ErrDuplicateMessageID   = errors.New("transformgen/model: duplicate message id")
)
