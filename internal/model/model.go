package model

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/wxdqing/go-transformgen/internal/define"
	"github.com/wxdqing/go-transformgen/internal/descriptor"
)

type Model struct {
	Modules []Module
}

type Module struct {
	Name          string
	ConstName     string
	InterfaceName string
	RPCs          []RPC
	Notifies      []Notify
}

type RPC struct {
	Method        string
	Ctx           string
	CtxImportPath string
	Request       descriptor.Message
	Response      descriptor.Message
}

type Notify struct {
	Method        string
	Ctx           string
	CtxImportPath string
	Message       descriptor.Message
}

func Build(desc *descriptor.Set, modules []define.Module) (*Model, error) {
	out := &Model{Modules: make([]Module, 0, len(modules))}
	seenRequests := make(map[uint32]string)
	for _, source := range modules {
		module := Module{
			Name:          source.Name,
			ConstName:     "ModelName" + pascal(source.Name),
			InterfaceName: pascal(source.Name),
		}
		for _, sourceRPC := range source.RPCs {
			req, err := requireMessage(desc, sourceRPC.Request, descriptor.MessageKindRequest)
			if err != nil {
				return nil, err
			}
			resp, err := requireMessage(desc, sourceRPC.Response, descriptor.MessageKindResponse)
			if err != nil {
				return nil, err
			}
			if previous, exists := seenRequests[req.ID]; exists {
				return nil, fmt.Errorf("%w: request %d bound by %s and %s", ErrDuplicateRequest, req.ID, previous, source.Name)
			}
			seenRequests[req.ID] = source.Name
			module.RPCs = append(module.RPCs, RPC{
				Method:        sourceRPC.Method,
				Ctx:           sourceRPC.Ctx,
				CtxImportPath: sourceRPC.CtxImportPath,
				Request:       req,
				Response:      resp,
			})
		}
		for _, sourceNotify := range source.Notifies {
			msg, err := requireMessage(desc, sourceNotify.Message, descriptor.MessageKindNotify)
			if err != nil {
				return nil, err
			}
			module.Notifies = append(module.Notifies, Notify{
				Method:        sourceNotify.Method,
				Ctx:           sourceNotify.Ctx,
				CtxImportPath: sourceNotify.CtxImportPath,
				Message:       msg,
			})
		}
		out.Modules = append(out.Modules, module)
	}
	return out, nil
}

func requireMessage(desc *descriptor.Set, fullName string, kind descriptor.MessageKind) (descriptor.Message, error) {
	msg, ok := desc.Message(fullName)
	if !ok {
		return descriptor.Message{}, fmt.Errorf("%w: %s", ErrMessageNotFound, fullName)
	}
	if msg.Kind != kind {
		return descriptor.Message{}, fmt.Errorf("%w: %s kind %d want %d", ErrMessageKindMismatch, fullName, msg.Kind, kind)
	}
	return msg, nil
}

func pascal(value string) string {
	parts := strings.Split(value, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		for i, r := range part {
			if i == 0 {
				builder.WriteRune(unicode.ToUpper(r))
				continue
			}
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
