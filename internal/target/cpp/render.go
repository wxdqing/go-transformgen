// Package cpptarget renders C++23 protocol bindings.
package cpptarget

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

// RuntimeMode controls how runtime abstractions are provided.
type RuntimeMode string

const (
	// RuntimeModeImport is reserved for a future external C++ runtime.
	RuntimeModeImport RuntimeMode = "import"
	// RuntimeModeEmit emits the transport-neutral C++ runtime header.
	RuntimeModeEmit RuntimeMode = "emit"
)

// Options configures C++ rendering.
type Options struct {
	Namespace          string
	Sides              []string
	Runtime            RuntimeMode
	ProtoIncludePrefix string
}

// File is one generated C++ source file.
type File struct {
	Path    string
	Content []byte
}

type selectedSides struct {
	requester bool
	responder bool
}

type messageView struct {
	ID       uint32
	EnumName string
	Kind     string
	FullName string
	CppType  string
	Include  string
}

type rpcView struct {
	Method       string
	RequestID    uint32
	ResponseID   uint32
	RequestEnum  string
	ResponseEnum string
	RequestType  string
	ResponseType string
}

type notifyView struct {
	Method      string
	MessageID   uint32
	MessageEnum string
	MessageType string
}

type moduleView struct {
	Namespace string
	Name      string
	RPCs      []rpcView
	Notifies  []notifyView
}

type protocolView struct {
	Namespace      string
	Includes       []string
	ServerMessages []messageView
	ClientMessages []messageView
	Messages       []messageView
}

// Render emits C++23 metadata, factories, runtime abstractions, and module bindings.
func Render(m *model.Model, opts Options) ([]File, error) {
	// Validate options before building any output.
	if opts.Namespace == "" {
		return nil, fmt.Errorf("cpp target: package namespace is required")
	}
	if opts.Runtime == "" {
		opts.Runtime = RuntimeModeEmit
	}
	if opts.Runtime != RuntimeModeEmit {
		return nil, fmt.Errorf("cpp target: only runtime emit is supported")
	}
	sides, err := parseSides(opts.Sides)
	if err != nil {
		return nil, err
	}

	// Build and validate the language-specific view.
	protocol, modules, err := buildView(m, opts)
	if err != nil {
		return nil, err
	}

	// Emit common protocol files.
	files := make([]File, 0, 3+len(modules)*4)
	for _, spec := range []struct {
		name string
		text string
		data any
	}{
		{name: "protocol_runtime.hpp", text: runtimeTemplate, data: protocol},
		{name: "protocol_messages.hpp", text: messagesHeaderTemplate, data: protocol},
		{name: "protocol_messages.cpp", text: messagesSourceTemplate, data: protocol},
	} {
		file, renderErr := renderTemplate(spec.name, spec.text, spec.data)
		if renderErr != nil {
			return nil, renderErr
		}
		files = append(files, file)
	}

	// Emit only the selected side for each YAML module.
	for _, module := range modules {
		if sides.requester {
			for _, spec := range []struct {
				suffix string
				text   string
			}{
				{suffix: "_requester.hpp", text: requesterHeaderTemplate},
				{suffix: "_requester.cpp", text: requesterSourceTemplate},
			} {
				file, renderErr := renderTemplate(module.Name+spec.suffix, spec.text, module)
				if renderErr != nil {
					return nil, renderErr
				}
				files = append(files, file)
			}
		}
		if sides.responder {
			for _, spec := range []struct {
				suffix string
				text   string
			}{
				{suffix: "_responder.hpp", text: responderHeaderTemplate},
				{suffix: "_responder.cpp", text: responderSourceTemplate},
			} {
				file, renderErr := renderTemplate(module.Name+spec.suffix, spec.text, module)
				if renderErr != nil {
					return nil, renderErr
				}
				files = append(files, file)
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// buildView converts the language-neutral model into deterministic C++ template data.
func buildView(m *model.Model, opts Options) (protocolView, []moduleView, error) {
	// Collect each referenced message exactly once.
	seen := make(map[string]messageView)
	includes := make(map[string]bool)
	modules := make([]moduleView, 0, len(m.Modules))
	for _, module := range m.Modules {
		mv := moduleView{Namespace: opts.Namespace, Name: module.Name}
		for _, rpc := range module.RPCs {
			req, err := cppMessage(rpc.Request, opts.ProtoIncludePrefix)
			if err != nil {
				return protocolView{}, nil, err
			}
			resp, err := cppMessage(rpc.Response, opts.ProtoIncludePrefix)
			if err != nil {
				return protocolView{}, nil, err
			}
			seen[req.FullName], seen[resp.FullName] = req, resp
			includes[req.Include], includes[resp.Include] = true, true
			mv.RPCs = append(mv.RPCs, rpcView{
				Method:       rpc.Method,
				RequestID:    req.ID,
				ResponseID:   resp.ID,
				RequestEnum:  req.EnumName,
				ResponseEnum: resp.EnumName,
				RequestType:  req.CppType,
				ResponseType: resp.CppType,
			})
		}
		for _, notify := range module.Notifies {
			msg, err := cppMessage(notify.Message, opts.ProtoIncludePrefix)
			if err != nil {
				return protocolView{}, nil, err
			}
			seen[msg.FullName] = msg
			includes[msg.Include] = true
			mv.Notifies = append(mv.Notifies, notifyView{
				Method:      notify.Method,
				MessageID:   msg.ID,
				MessageEnum: msg.EnumName,
				MessageType: msg.CppType,
			})
		}
		modules = append(modules, mv)
	}

	// Sort common metadata and includes for stable generated files.
	view := protocolView{Namespace: opts.Namespace}
	for include := range includes {
		view.Includes = append(view.Includes, include)
	}
	for _, msg := range seen {
		view.Messages = append(view.Messages, msg)
		if msg.Kind == "Request" {
			view.ServerMessages = append(view.ServerMessages, msg)
		} else {
			view.ClientMessages = append(view.ClientMessages, msg)
		}
	}
	sort.Strings(view.Includes)
	sort.Slice(view.Messages, func(i, j int) bool { return view.Messages[i].ID < view.Messages[j].ID })
	sort.Slice(view.ServerMessages, func(i, j int) bool { return view.ServerMessages[i].ID < view.ServerMessages[j].ID })
	sort.Slice(view.ClientMessages, func(i, j int) bool { return view.ClientMessages[i].ID < view.ClientMessages[j].ID })
	sort.Slice(modules, func(i, j int) bool { return modules[i].Name < modules[j].Name })
	return view, modules, nil
}

// cppMessage maps a top-level protobuf message to its official protoc C++ type and header.
func cppMessage(msg descriptor.Message, prefix string) (messageView, error) {
	// Reject nested messages until their generated C++ naming can be represented explicitly.
	relative := msg.FullName
	if msg.ProtoPackage != "" {
		expected := msg.ProtoPackage + "."
		if !strings.HasPrefix(relative, expected) {
			return messageView{}, fmt.Errorf("cpp target: message %s is outside proto package %s", msg.FullName, msg.ProtoPackage)
		}
		relative = strings.TrimPrefix(relative, expected)
	}
	if strings.Contains(relative, ".") {
		return messageView{}, fmt.Errorf("cpp target: nested RPC/notify message %s is not supported", msg.FullName)
	}

	// Build the protoc namespace and flattened protobuf header include.
	parts := make([]string, 0, 2)
	if msg.ProtoPackage != "" {
		parts = append(parts, strings.Split(msg.ProtoPackage, ".")...)
	}
	parts = append(parts, msg.ProtoName)
	stem := strings.TrimSuffix(filepath.Base(msg.SourceFile), filepath.Ext(msg.SourceFile))
	include := stem + ".pb.h"
	if cleanPrefix := strings.Trim(strings.ReplaceAll(prefix, "\\", "/"), "/"); cleanPrefix != "" {
		include = cleanPrefix + "/" + include
	}
	return messageView{
		ID:       msg.ID,
		EnumName: msg.ProtoName,
		Kind:     cppKind(msg.Kind),
		FullName: msg.FullName,
		CppType:  "::" + strings.Join(parts, "::"),
		Include:  include,
	}, nil
}

// cppKind returns the generated C++ metadata enum member.
func cppKind(kind descriptor.MessageKind) string {
	switch kind {
	case descriptor.MessageKindRequest:
		return "Request"
	case descriptor.MessageKindResponse:
		return "Response"
	case descriptor.MessageKindNotify:
		return "Notify"
	default:
		return "Unknown"
	}
}

// parseSides validates requester/responder selection.
func parseSides(values []string) (selectedSides, error) {
	// Empty selection means both generated sides.
	if len(values) == 0 {
		return selectedSides{requester: true, responder: true}, nil
	}
	var sides selectedSides
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "requester":
			sides.requester = true
		case "responder":
			sides.responder = true
		case "":
		default:
			return selectedSides{}, fmt.Errorf("cpp target: unsupported side %q", value)
		}
	}
	if !sides.requester && !sides.responder {
		return selectedSides{}, fmt.Errorf("cpp target: at least one side is required")
	}
	return sides, nil
}

// renderTemplate executes one embedded C++ template.
func renderTemplate(name, text string, data any) (File, error) {
	// Parse and execute without applying a C++ formatter.
	tmpl, err := template.New(name).Parse(text)
	if err != nil {
		return File{}, fmt.Errorf("cpp target: parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return File{}, fmt.Errorf("cpp target: render %s: %w", name, err)
	}
	return File{Path: name, Content: buf.Bytes()}, nil
}

//go:embed templates/protocol_runtime.hpp.tmpl
var runtimeTemplate string

//go:embed templates/protocol_messages.hpp.tmpl
var messagesHeaderTemplate string

//go:embed templates/protocol_messages.cpp.tmpl
var messagesSourceTemplate string

//go:embed templates/module_requester.hpp.tmpl
var requesterHeaderTemplate string

//go:embed templates/module_requester.cpp.tmpl
var requesterSourceTemplate string

//go:embed templates/module_responder.hpp.tmpl
var responderHeaderTemplate string

//go:embed templates/module_responder.cpp.tmpl
var responderSourceTemplate string
