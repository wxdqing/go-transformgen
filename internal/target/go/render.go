package gotarget

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

type Options struct {
	Package string
	Sides   []string
}

type File struct {
	Path    string
	Content []byte
}

func Render(m *model.Model, opts Options) ([]File, error) {
	if opts.Package == "" {
		return nil, fmt.Errorf("go target: package is required")
	}
	view := buildView(m, opts)

	protocol, err := renderFormatted("protocol.go", protocolTemplate, view)
	if err != nil {
		return nil, err
	}
	messages, err := renderFormatted("protocol_messages.go", messagesTemplate, view)
	if err != nil {
		return nil, err
	}
	files := []File{
		{Path: "protocol.go", Content: protocol},
		{Path: "protocol_messages.go", Content: messages},
	}
	for _, module := range view.Modules {
		content, err := renderFormatted(module.Name+".go", moduleTemplate, module)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: module.Name + ".go", Content: content})
	}
	return files, nil
}

type viewModel struct {
	Package  string
	Imports  []importView
	Modules  []moduleView
	Messages []messageView
	RPCs     []rpcView
	Notifies []notifyView
}

type moduleView struct {
	Package       string
	Imports       []importView
	Name          string
	ConstName     string
	InterfaceName string
	RPCs          []rpcView
	Notifies      []notifyView
}

type messageView struct {
	ConstName  string
	ID         uint32
	Kind       string
	KindSuffix string
	FullName   string
	GoType     string
}

type importView struct {
	Name string
	Path string
}

type rpcView struct {
	Method        string
	Ctx           string
	CtxImportPath string
	ModelConst    string
	RequestConst  string
	ResponseConst string
	RequestType   string
	ResponseType  string
}

type notifyView struct {
	Method        string
	Ctx           string
	CtxImportPath string
	ModelConst    string
	MessageConst  string
	MessageType   string
}

func buildView(m *model.Model, opts Options) viewModel {
	seen := make(map[uint32]messageView)
	imports := make(map[string]importView)
	view := viewModel{Package: opts.Package}
	for _, module := range m.Modules {
		moduleImports := make(map[string]importView)
		mv := moduleView{
			Package:       opts.Package,
			Name:          module.Name,
			ConstName:     module.ConstName,
			InterfaceName: module.InterfaceName,
		}
		for _, rpc := range module.RPCs {
			req := messageFromDescriptor(rpc.Request, opts.Package)
			resp := messageFromDescriptor(rpc.Response, opts.Package)
			seen[rpc.Request.ID] = req
			seen[rpc.Response.ID] = resp
			addImport(imports, rpc.Request, opts.Package)
			addImport(imports, rpc.Response, opts.Package)
			addImport(moduleImports, rpc.Request, opts.Package)
			addImport(moduleImports, rpc.Response, opts.Package)
			addCtxImport(moduleImports, rpc.Ctx, rpc.CtxImportPath)
			mv.RPCs = append(mv.RPCs, rpcView{
				Method:        rpc.Method,
				Ctx:           rpc.Ctx,
				CtxImportPath: rpc.CtxImportPath,
				ModelConst:    module.ConstName,
				RequestConst:  req.ConstName,
				ResponseConst: resp.ConstName,
				RequestType:   req.GoType,
				ResponseType:  resp.GoType,
			})
			view.RPCs = append(view.RPCs, mv.RPCs[len(mv.RPCs)-1])
		}
		for _, notify := range module.Notifies {
			msg := messageFromDescriptor(notify.Message, opts.Package)
			seen[notify.Message.ID] = msg
			addImport(imports, notify.Message, opts.Package)
			addImport(moduleImports, notify.Message, opts.Package)
			addCtxImport(moduleImports, notify.Ctx, notify.CtxImportPath)
			mv.Notifies = append(mv.Notifies, notifyView{
				Method:        notify.Method,
				Ctx:           notify.Ctx,
				CtxImportPath: notify.CtxImportPath,
				ModelConst:    module.ConstName,
				MessageConst:  msg.ConstName,
				MessageType:   msg.GoType,
			})
			view.Notifies = append(view.Notifies, mv.Notifies[len(mv.Notifies)-1])
		}
		mv.Imports = sortedImports(moduleImports)
		view.Modules = append(view.Modules, mv)
	}
	for _, msg := range seen {
		view.Messages = append(view.Messages, msg)
	}
	view.Imports = sortedImports(imports)
	sort.Slice(view.Modules, func(i, j int) bool {
		return view.Modules[i].Name < view.Modules[j].Name
	})
	sort.Slice(view.Messages, func(i, j int) bool {
		return view.Messages[i].ID < view.Messages[j].ID
	})
	return view
}

func messageFromDescriptor(msg descriptor.Message, packageName string) messageView {
	return messageView{
		ConstName:  "MessageID" + msg.GoTypeName,
		ID:         msg.ID,
		Kind:       kindName(msg.Kind),
		KindSuffix: kindSuffix(msg.Kind),
		FullName:   msg.FullName,
		GoType:     goTypeRef(msg, packageName),
	}
}

func goTypeRef(msg descriptor.Message, packageName string) string {
	if msg.GoImportPath == "" || msg.GoPackageName == "" || msg.GoPackageName == packageName {
		return msg.GoTypeName
	}
	return msg.GoPackageName + "." + msg.GoTypeName
}

func addImport(imports map[string]importView, msg descriptor.Message, packageName string) {
	if msg.GoImportPath == "" || msg.GoPackageName == "" || msg.GoPackageName == packageName {
		return
	}
	imports[msg.GoImportPath] = importView{Name: msg.GoPackageName, Path: msg.GoImportPath}
}

func addCtxImport(imports map[string]importView, ctx string, importPath string) {
	qualifier := qualifierName(ctx)
	if qualifier == "" {
		return
	}
	if importPath == "" {
		if qualifier != "context" {
			return
		}
		importPath = "context"
	}
	name := ""
	if base := path.Base(importPath); base != qualifier {
		name = qualifier
	}
	imports[importPath] = importView{Name: name, Path: importPath}
}

func qualifierName(typeName string) string {
	idx := strings.LastIndex(typeName, ".")
	if idx <= 0 {
		return ""
	}
	prefix := typeName[:idx]
	if strings.ContainsAny(prefix, "[]* ") {
		return ""
	}
	return prefix
}

func sortedImports(imports map[string]importView) []importView {
	out := make([]importView, 0, len(imports))
	for _, value := range imports {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func kindName(kind descriptor.MessageKind) string {
	switch kind {
	case descriptor.MessageKindRequest:
		return "MessageKindRequest"
	case descriptor.MessageKindResponse:
		return "MessageKindResponse"
	case descriptor.MessageKindNotify:
		return "MessageKindNotify"
	default:
		return "0"
	}
}

func kindSuffix(kind descriptor.MessageKind) string {
	switch kind {
	case descriptor.MessageKindRequest:
		return "Request"
	case descriptor.MessageKindResponse:
		return "Response"
	case descriptor.MessageKindNotify:
		return "Notify"
	default:
		return ""
	}
}

func executeTemplate(name, text string, data any) ([]byte, error) {
	tmpl, err := template.New(name).Parse(text)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderFormatted(name, text string, data any) ([]byte, error) {
	raw, err := executeTemplate(name, text, data)
	if err != nil {
		return nil, err
	}
	formatted, err := format.Source(raw)
	if err != nil {
		return nil, fmt.Errorf("%w\n%s", err, raw)
	}
	return formatted, nil
}

//go:embed templates/messages.go.tmpl
var messagesTemplate string

//go:embed templates/module.go.tmpl
var moduleTemplate string

//go:embed templates/protocol.go.tmpl
var protocolTemplate string
