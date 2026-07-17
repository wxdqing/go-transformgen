package csharptarget

import (
	"bytes"
	_ "embed"
	"fmt"
	"path/filepath"
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

// Options keeps the CLI surface stable. Generated C# lives in the global
// namespace to match TianLong3 protobuf-net types, so Namespace and Sides are
// accepted but ignored.
type Options struct {
	Namespace string
	Sides     []string
	Runtime   RuntimeMode
}

type File struct {
	Path    string
	Content []byte
}

// Render emits TianLong3-style client protocol files:
//   - per-proto protobuf-net message/enum classes
//   - EMsgToServerType / EMsgToClientType / EMsgType bindings
func Render(m *model.Model, desc *descriptor.Set, opts Options) ([]File, error) {
	if opts.Runtime == "" {
		opts.Runtime = RuntimeModeEmit
	}
	if opts.Runtime != RuntimeModeEmit {
		return nil, fmt.Errorf("csharp target: only runtime emit is supported")
	}
	if desc == nil {
		return nil, fmt.Errorf("csharp target: descriptor set is required")
	}
	protocol := buildProtocolView(m)
	files := make([]File, 0, 3+len(desc.Files()))
	for _, spec := range []struct {
		name string
		text string
		data any
	}{
		{name: "EMsgToServerType.cs", text: serverEnumTemplate, data: protocol},
		{name: "EMsgToClientType.cs", text: clientEnumTemplate, data: protocol},
		{name: "EMsgType.cs", text: bindingTemplate, data: protocol},
	} {
		file, err := renderTemplate(spec.name, spec.text, spec.data)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	for _, protoFile := range buildProtoFileViews(desc) {
		if len(protoFile.Messages) == 0 && len(protoFile.Enums) == 0 {
			continue
		}
		file, err := renderTemplate(protoFile.OutputName, protoFileTemplate, protoFile)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

type enumMember struct {
	Name string
	ID   uint32
}

type binding struct {
	ClassName   string
	Interface   string
	EnumType    string
	EnumName    string
	HasRetError bool
}

type protocolView struct {
	ServerMessages []enumMember
	ClientMessages []enumMember
	Bindings       []binding
	HasAnyRetError bool
}

type fieldView struct {
	Number             int32
	Property           string
	Type               string
	MapKeyType         string
	MapValType         string
	IsRepeated         bool
	IsString           bool
	IsMap              bool
	IsGroup            bool
	IsOneof            bool
	IsFirstOneofMember bool
	OneofUnionType     string
	OneofUnionField    string
	OneofGetter        string
	OneofSetter        string
}

type oneofMemberView struct {
	Property string
	Number   int32
}

type oneofView struct {
	CaseEnum     string
	CaseProperty string
	UnionField   string
	Members      []oneofMemberView
}

type messageView struct {
	Name   string
	Fields []fieldView
	Oneofs []oneofView
}

type enumView struct {
	Name   string
	Values []descriptor.EnumValue
}

type protoFileView struct {
	BaseName   string
	OutputName string
	Messages   []messageView
	Enums      []enumView
}

func buildProtocolView(m *model.Model) protocolView {
	seen := make(map[string]bool)
	var view protocolView
	add := func(msg descriptor.Message) {
		if msg.FullName == "" || seen[msg.FullName] {
			return
		}
		seen[msg.FullName] = true
		name := msg.CSharp.TypeName
		if name == "" {
			name = msg.ProtoName
		}
		toServer := msg.Kind == descriptor.MessageKindRequest
		if msg.HasRetError {
			view.HasAnyRetError = true
		}
		if toServer {
			view.ServerMessages = append(view.ServerMessages, enumMember{Name: name, ID: msg.ID})
			view.Bindings = append(view.Bindings, binding{
				ClassName:   name,
				Interface:   "IProtoBufToServer",
				EnumType:    "EMsgToServerType",
				EnumName:    name,
				HasRetError: msg.HasRetError,
			})
			return
		}
		view.ClientMessages = append(view.ClientMessages, enumMember{Name: name, ID: msg.ID})
		view.Bindings = append(view.Bindings, binding{
			ClassName:   name,
			Interface:   "IProtoBufToClient",
			EnumType:    "EMsgToClientType",
			EnumName:    name,
			HasRetError: msg.HasRetError,
		})
	}
	for _, module := range m.Modules {
		for _, rpc := range module.RPCs {
			add(rpc.Request)
			add(rpc.Response)
		}
		for _, notify := range module.Notifies {
			add(notify.Message)
		}
	}
	sort.Slice(view.ServerMessages, func(i, j int) bool { return view.ServerMessages[i].ID < view.ServerMessages[j].ID })
	sort.Slice(view.ClientMessages, func(i, j int) bool { return view.ClientMessages[i].ID < view.ClientMessages[j].ID })
	sort.Slice(view.Bindings, func(i, j int) bool { return view.Bindings[i].ClassName < view.Bindings[j].ClassName })
	return view
}

func buildProtoFileViews(desc *descriptor.Set) []protoFileView {
	files := desc.Files()
	out := make([]protoFileView, 0, len(files))
	for _, file := range files {
		view := protoFileView{
			BaseName:   file.BaseName,
			OutputName: csharpFileName(file.BaseName),
		}
		for _, fullName := range file.Enums {
			enum, ok := desc.Enum(fullName)
			if !ok {
				continue
			}
			view.Enums = append(view.Enums, enumView{Name: enum.ProtoName, Values: enum.Values})
		}
		for _, fullName := range file.Messages {
			msg, ok := desc.Message(fullName)
			if !ok {
				continue
			}
			mv := messageView{Name: msg.ProtoName}
			for _, field := range msg.Fields {
				mv.Fields = append(mv.Fields, fieldViewFromDescriptor(field))
			}
			for _, oneof := range msg.Oneofs {
				ov := oneofView{
					CaseEnum:     pascal(oneof.Name) + "OneofCase",
					CaseProperty: pascal(oneof.Name) + "Case",
					UnionField:   "__pbn__" + oneof.Name,
				}
				for _, field := range msg.Fields {
					if field.OneofName != oneof.Name {
						continue
					}
					ov.Members = append(ov.Members, oneofMemberView{
						Property: csharpPropertyName(field),
						Number:   field.Number,
					})
				}
				mv.Oneofs = append(mv.Oneofs, ov)
			}
			view.Messages = append(view.Messages, mv)
		}
		out = append(out, view)
	}
	return out
}

func fieldViewFromDescriptor(field descriptor.Field) fieldView {
	typeName := field.ScalarType
	switch field.TypeKind {
	case descriptor.FieldTypeMessage, descriptor.FieldTypeEnum:
		typeName = field.TypeName
	case descriptor.FieldTypeBytes:
		typeName = "byte[]"
	case descriptor.FieldTypeMap:
		typeName = fmt.Sprintf("Dictionary<%s, %s>", field.MapKeyType, field.MapValType)
	}
	view := fieldView{
		Number:             field.Number,
		Property:           csharpPropertyName(field),
		Type:               typeName,
		MapKeyType:         field.MapKeyType,
		MapValType:         field.MapValType,
		IsRepeated:         field.Cardinal == descriptor.FieldRepeated && field.TypeKind != descriptor.FieldTypeMap && field.OneofName == "",
		IsString:           field.TypeKind == descriptor.FieldTypeScalar && field.ScalarType == "string" && field.Cardinal != descriptor.FieldRepeated && field.OneofName == "",
		IsMap:              field.TypeKind == descriptor.FieldTypeMap,
		IsGroup:            field.IsGroup,
		IsOneof:            field.OneofName != "",
		IsFirstOneofMember: field.IsFirstOneofMember,
		OneofUnionType:     field.OneofUnionType,
		OneofUnionField:    field.OneofUnionField,
	}
	if view.IsOneof {
		view.OneofGetter, view.OneofSetter = oneofAccessors(field, typeName)
	}
	return view
}

func oneofAccessors(field descriptor.Field, typeName string) (getter, setter string) {
	storage := field.OneofStorage
	union := field.OneofUnionField
	switch {
	case field.TypeKind == descriptor.FieldTypeEnum:
		return fmt.Sprintf("((%s)%s.%s)", typeName, union, storage), fmt.Sprintf("(int)value")
	case storage == "Object":
		return fmt.Sprintf("((%s)%s.Object)", typeName, union), "value"
	default:
		return fmt.Sprintf("%s.%s", union, storage), "value"
	}
}

func csharpPropertyName(field descriptor.Field) string {
	name := field.JSONName
	if name == "" {
		name = field.Name
	}
	if name == "" {
		return name
	}
	if strings.Contains(name, "_") {
		return pascal(name)
	}
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func csharpFileName(baseName string) string {
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	return pascal(stem) + ".cs"
}

func pascal(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
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

//go:embed templates/EMsgToServerType.cs.tmpl
var serverEnumTemplate string

//go:embed templates/EMsgToClientType.cs.tmpl
var clientEnumTemplate string

//go:embed templates/EMsgType.cs.tmpl
var bindingTemplate string

//go:embed templates/ProtoFile.cs.tmpl
var protoFileTemplate string
