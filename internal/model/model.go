package model

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/wxdqing/go-transformgen/internal/define"
	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/msgid"
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

// Build links YAML modules to descriptor messages and assigns wire ids.
// locked maps message short names to previously assigned ids (from msgid.lock.yaml).
// The returned map is the synced lock contents: only current messages, ready to save.
// Names present in locked keep their ids; absent names are hashed with upward probing
// against ids still in use. Deleted lock entries are omitted so those ids are reusable.
func Build(desc *descriptor.Set, modules []define.Module, locked map[string]uint32) (*Model, map[string]uint32, error) {
	out := &Model{Modules: make([]Module, 0, len(modules))}
	seenRequests := make(map[string]string)
	for _, source := range modules {
		module := Module{
			Name:          source.Name,
			ConstName:     "ModelName" + pascal(source.Name),
			InterfaceName: pascal(source.Name),
		}
		for _, sourceRPC := range source.RPCs {
			req, err := requireMessage(desc, sourceRPC.Request, descriptor.MessageKindRequest)
			if err != nil {
				return nil, nil, err
			}
			resp, err := requireMessage(desc, sourceRPC.Response, descriptor.MessageKindResponse)
			if err != nil {
				return nil, nil, err
			}
			if previous, exists := seenRequests[req.FullName]; exists {
				return nil, nil, fmt.Errorf("%w: request %s bound by %s and %s", ErrDuplicateRequest, req.FullName, previous, source.Name)
			}
			seenRequests[req.FullName] = source.Name
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
				return nil, nil, err
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
	nextLock, err := assignMessageIDs(out, locked)
	if err != nil {
		return nil, nil, err
	}
	return out, nextLock, nil
}

type idEntry struct {
	full     string
	name     string
	toServer bool
}

func collectIDEntries(out *Model) ([]idEntry, error) {
	seenFull := make(map[string]bool)
	byName := make(map[string]string)
	var entries []idEntry
	add := func(m descriptor.Message) error {
		if m.FullName == "" || seenFull[m.FullName] {
			return nil
		}
		seenFull[m.FullName] = true
		if previous, exists := byName[m.ProtoName]; exists && previous != m.FullName {
			return fmt.Errorf("%w: short name %s used by %s and %s", ErrDuplicateMessageName, m.ProtoName, previous, m.FullName)
		}
		byName[m.ProtoName] = m.FullName
		entries = append(entries, idEntry{
			full:     m.FullName,
			name:     m.ProtoName,
			toServer: m.Kind == descriptor.MessageKindRequest,
		})
		return nil
	}
	for _, module := range out.Modules {
		for _, rpc := range module.RPCs {
			if err := add(rpc.Request); err != nil {
				return nil, err
			}
			if err := add(rpc.Response); err != nil {
				return nil, err
			}
		}
		for _, notify := range module.Notifies {
			if err := add(notify.Message); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, nil
}

// assignMessageIDs prefers locked ids for known short names. New messages get a
// deterministic hash id, probing upward past ids still reserved by the current set.
func assignMessageIDs(out *Model, locked map[string]uint32) (map[string]uint32, error) {
	entries, err := collectIDEntries(out)
	if err != nil {
		return nil, err
	}
	if locked == nil {
		locked = map[string]uint32{}
	}

	used := make(map[uint32]string, len(entries))
	finalByFull := make(map[string]uint32, len(entries))
	nextLock := make(map[string]uint32, len(entries))

	// First pass: reinstate locked ids for messages that still exist.
	for _, e := range entries {
		id, ok := locked[e.name]
		if !ok {
			continue
		}
		if owner, taken := used[id]; taken {
			return nil, fmt.Errorf("%w: id %d locked by %s and %s", ErrDuplicateMessageID, id, owner, e.name)
		}
		used[id] = e.name
		finalByFull[e.full] = id
		nextLock[e.name] = id
	}

	// Second pass: assign fresh ids for messages absent from the lock.
	for _, e := range entries {
		if _, ok := nextLock[e.name]; ok {
			continue
		}
		id := msgid.Compute(e.name, e.toServer)
		for used[id] != "" {
			id++
		}
		used[id] = e.name
		finalByFull[e.full] = id
		nextLock[e.name] = id
	}

	for mi := range out.Modules {
		module := &out.Modules[mi]
		for ri := range module.RPCs {
			module.RPCs[ri].Request.ID = finalByFull[module.RPCs[ri].Request.FullName]
			module.RPCs[ri].Response.ID = finalByFull[module.RPCs[ri].Response.FullName]
		}
		for ni := range module.Notifies {
			module.Notifies[ni].Message.ID = finalByFull[module.Notifies[ni].Message.FullName]
		}
	}
	return nextLock, nil
}

func requireMessage(desc *descriptor.Set, fullName string, kind descriptor.MessageKind) (descriptor.Message, error) {
	msg, ok := desc.Message(fullName)
	if !ok {
		return descriptor.Message{}, fmt.Errorf("%w: %s", ErrMessageNotFound, fullName)
	}
	// YAML role is authoritative. When the name also implies a kind, require them to match.
	if inferred, ok := descriptor.InferKindFromName(msg.ProtoName); ok && inferred != kind {
		return descriptor.Message{}, fmt.Errorf("%w: %s kind from name %d want %d", ErrMessageKindMismatch, fullName, inferred, kind)
	}
	if msg.Kind != 0 && msg.Kind != kind {
		return descriptor.Message{}, fmt.Errorf("%w: %s kind %d want %d", ErrMessageKindMismatch, fullName, msg.Kind, kind)
	}
	msg.Kind = kind
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
