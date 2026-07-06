package csharptarget

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

type RuntimeMode string

const (
	RuntimeModeImport RuntimeMode = "import"
	RuntimeModeEmit   RuntimeMode = "emit"
)

type Options struct {
	Namespace string
	Sides     []string
	Runtime   RuntimeMode
}

type File struct {
	Path    string
	Content []byte
}

func Render(m *model.Model, opts Options) ([]File, error) {
	if opts.Namespace == "" {
		return nil, fmt.Errorf("csharp target: namespace is required")
	}
	if opts.Runtime == "" {
		opts.Runtime = RuntimeModeEmit
	}
	if opts.Runtime != RuntimeModeEmit {
		return nil, fmt.Errorf("csharp target: only runtime emit is supported")
	}
	side, err := parseSides(opts.Sides)
	if err != nil {
		return nil, err
	}
	view := buildView(m, opts, side)
	files := make([]File, 0, 3+len(view.Modules)*2)
	for _, spec := range []struct {
		name string
		text string
	}{
		{name: "Frame.cs", text: frameTemplate},
		{name: "ProtocolRuntime.cs", text: runtimeTemplate},
		{name: "ProtocolMessages.cs", text: messagesTemplate},
	} {
		file, err := renderTemplate(spec.name, spec.text, view)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	for _, module := range view.Modules {
		if side.requester {
			file, err := renderTemplate(module.ClassName+"Requester.cs", requesterTemplate, module)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
		if side.responder {
			file, err := renderTemplate(module.ClassName+"Responder.cs", responderTemplate, module)
			if err != nil {
				return nil, err
			}
			files = append(files, file)
		}
	}
	return files, nil
}

type selectedSides struct {
	requester bool
	responder bool
}

func parseSides(sides []string) (selectedSides, error) {
	if len(sides) == 0 {
		return selectedSides{requester: true}, nil
	}
	var out selectedSides
	for _, side := range sides {
		switch strings.TrimSpace(side) {
		case "requester":
			out.requester = true
		case "responder":
			out.responder = true
		case "":
		default:
			return selectedSides{}, fmt.Errorf("csharp target: unsupported side %q", side)
		}
	}
	if !out.requester && !out.responder {
		return selectedSides{}, fmt.Errorf("csharp target: at least one side is required")
	}
	return out, nil
}

type viewModel struct {
	Namespace string
	Messages  []messageView
	Modules   []moduleView
}

type moduleView struct {
	Namespace string
	ClassName string
	RPCs      []rpcView
	Notifies  []notifyView
}

type messageView struct {
	Name      string
	ConstName string
	ID        uint32
	Kind      descriptor.MessageKind
	Type      string
}

type rpcView struct {
	Method       string
	RequestID    string
	ResponseID   string
	RequestType  string
	ResponseType string
}

type notifyView struct {
	Method    string
	MessageID string
	Type      string
}

func buildView(m *model.Model, opts Options, _ selectedSides) viewModel {
	seen := make(map[uint32]messageView)
	view := viewModel{Namespace: opts.Namespace}
	for _, module := range m.Modules {
		mv := moduleView{Namespace: opts.Namespace, ClassName: pascal(module.Name)}
		for _, rpc := range module.RPCs {
			req := messageFromDescriptor(rpc.Request, opts.Namespace)
			resp := messageFromDescriptor(rpc.Response, opts.Namespace)
			seen[rpc.Request.ID] = req
			seen[rpc.Response.ID] = resp
			mv.RPCs = append(mv.RPCs, rpcView{
				Method:       rpc.Method,
				RequestID:    req.ConstName,
				ResponseID:   resp.ConstName,
				RequestType:  req.Type,
				ResponseType: resp.Type,
			})
		}
		for _, notify := range module.Notifies {
			msg := messageFromDescriptor(notify.Message, opts.Namespace)
			seen[notify.Message.ID] = msg
			mv.Notifies = append(mv.Notifies, notifyView{
				Method:    notify.Method,
				MessageID: msg.ConstName,
				Type:      msg.Type,
			})
		}
		view.Modules = append(view.Modules, mv)
	}
	for _, msg := range seen {
		view.Messages = append(view.Messages, msg)
	}
	sort.Slice(view.Messages, func(i, j int) bool {
		return view.Messages[i].ID < view.Messages[j].ID
	})
	sort.Slice(view.Modules, func(i, j int) bool {
		return view.Modules[i].ClassName < view.Modules[j].ClassName
	})
	return view
}

func messageFromDescriptor(msg descriptor.Message, namespace string) messageView {
	typeName := msg.CSharp.TypeName
	if typeName == "" {
		typeName = msg.ProtoName
	}
	if typeName == "" {
		typeName = msg.GoTypeName
	}
	ref := typeName
	if msg.CSharp.Namespace != "" && msg.CSharp.Namespace != namespace {
		ref = msg.CSharp.Namespace + "." + typeName
	}
	return messageView{
		Name:      typeName,
		ConstName: typeName,
		ID:        msg.ID,
		Kind:      msg.Kind,
		Type:      ref,
	}
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

func renderTemplate(name, text string, data any) (File, error) {
	tmpl, err := template.New(name).Parse(text)
	if err != nil {
		return File{}, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return File{}, err
	}
	return File{Path: name, Content: buf.Bytes()}, nil
}

//go:embed templates/Frame.cs.tmpl
var frameTemplate string

//go:embed templates/ProtocolRuntime.cs.tmpl
var runtimeTemplate string

//go:embed templates/ProtocolMessages.cs.tmpl
var messagesTemplate string

//go:embed templates/Requester.cs.tmpl
var requesterTemplate string

//go:embed templates/Responder.cs.tmpl
var responderTemplate string
