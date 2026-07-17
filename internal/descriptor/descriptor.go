package descriptor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type MessageKind uint8

const (
	MessageKindRequest  MessageKind = 1
	MessageKindResponse MessageKind = 2
	MessageKindNotify   MessageKind = 3
)

type FieldCardinal uint8

const (
	FieldSingular FieldCardinal = 1
	FieldRepeated FieldCardinal = 2
)

type FieldTypeKind uint8

const (
	FieldTypeScalar  FieldTypeKind = 1
	FieldTypeMessage FieldTypeKind = 2
	FieldTypeEnum    FieldTypeKind = 3
	FieldTypeBytes   FieldTypeKind = 4
	FieldTypeMap     FieldTypeKind = 5
)

type Field struct {
	Number     int32
	Name       string // proto field name (snake_case)
	JSONName   string
	Cardinal   FieldCardinal
	TypeKind   FieldTypeKind
	ScalarType string // C# scalar type when TypeKind == Scalar
	TypeName   string // short message/enum type name when applicable
	FullType   string // fully-qualified proto type without leading dot
	IsGroup    bool   // proto2 group field (message with group wire encoding)
	MapKeyType string // C# key type when TypeKind == Map
	MapValType string // C# value type when TypeKind == Map

	// Oneof (real oneof only; proto3 optional synthetic oneofs are ignored).
	OneofName          string
	OneofUnionType     string // e.g. DiscriminatedUnionObject
	OneofUnionField    string // e.g. __pbn__payload
	OneofStorage       string // e.g. Object, Int32
	IsFirstOneofMember bool
}

// Oneof describes a real (non-synthetic) oneof group on a message.
type Oneof struct {
	Name      string
	UnionType string
	Fields    []string // member field names in declaration order
}

type EnumValue struct {
	Name   string
	Number int32
}

type Enum struct {
	FullName     string
	ProtoPackage string
	ProtoName    string
	Values       []EnumValue
	SourceFile   string
}

type Message struct {
	ID           uint32
	Kind         MessageKind
	FullName     string
	ProtoPackage string
	ProtoName    string
	SourceFile   string
	Fields       []Field
	Oneofs       []Oneof
	HasRetError  bool
	Go           GoMessage
	CSharp       CSharpMessage

	GoImportPath  string
	GoPackageName string
	GoTypeName    string
}

type GoMessage struct {
	ImportPath  string
	PackageName string
	TypeName    string
}

type CSharpMessage struct {
	Namespace string
	TypeName  string
}

type File struct {
	Name            string // descriptor path, e.g. example/transform/heartbeat.proto
	BaseName        string // heartbeat.proto
	ProtoPackage    string
	GoImportPath    string
	GoPackageName   string
	CsharpNamespace string
	Messages        []string // full names in declaration order
	Enums           []string // full names in declaration order
}

type Set struct {
	messages map[string]Message
	enums    map[string]Enum
	files    []File
}

func NewSet(messages ...Message) *Set {
	out := &Set{
		messages: make(map[string]Message),
		enums:    make(map[string]Enum),
	}
	for _, msg := range messages {
		out.messages[msg.FullName] = msg
	}
	return out
}

// NewSetWithFiles builds a descriptor set for tests that exercise file-scoped
// C# rendering (messages + enums grouped by proto file).
func NewSetWithFiles(files []File, enums []Enum, messages []Message) *Set {
	out := &Set{
		messages: make(map[string]Message),
		enums:    make(map[string]Enum),
		files:    append([]File(nil), files...),
	}
	for _, e := range enums {
		out.enums[e.FullName] = e
	}
	for _, msg := range messages {
		out.messages[msg.FullName] = msg
	}
	return out
}

func Load(path string) (*Set, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var files descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &files); err != nil {
		return nil, err
	}
	out := &Set{
		messages: make(map[string]Message),
		enums:    make(map[string]Enum),
	}
	for _, file := range files.GetFile() {
		pkg := file.GetPackage()
		if skipPackage(pkg) {
			continue
		}
		goImport, goPackage := splitGoPackage(file.GetOptions().GetGoPackage())
		csharpNamespace := file.GetOptions().GetCsharpNamespace()
		sourceFile := file.GetName()
		fileInfo := File{
			Name:            sourceFile,
			BaseName:        filepath.Base(sourceFile),
			ProtoPackage:    pkg,
			GoImportPath:    goImport,
			GoPackageName:   goPackage,
			CsharpNamespace: csharpNamespace,
		}
		for _, enum := range file.GetEnumType() {
			e, err := readEnum(pkg, sourceFile, enum)
			if err != nil {
				return nil, err
			}
			if _, ok := out.enums[e.FullName]; ok {
				return nil, fmt.Errorf("%w: %s", ErrDuplicateMessage, e.FullName)
			}
			out.enums[e.FullName] = e
			fileInfo.Enums = append(fileInfo.Enums, e.FullName)
		}
		for _, msg := range file.GetMessageType() {
			if err := collectMessage(out, &fileInfo, pkg, goImport, goPackage, csharpNamespace, sourceFile, "", msg); err != nil {
				return nil, err
			}
		}
		out.files = append(out.files, fileInfo)
	}
	return out, nil
}

func (s *Set) Message(fullName string) (Message, bool) {
	if s == nil {
		return Message{}, false
	}
	msg, ok := s.messages[fullName]
	return msg, ok
}

func (s *Set) Messages() []Message {
	if s == nil {
		return nil
	}
	out := make([]Message, 0, len(s.messages))
	for _, msg := range s.messages {
		out = append(out, msg)
	}
	return out
}

func (s *Set) Enum(fullName string) (Enum, bool) {
	if s == nil {
		return Enum{}, false
	}
	e, ok := s.enums[fullName]
	return e, ok
}

func (s *Set) Enums() []Enum {
	if s == nil {
		return nil
	}
	out := make([]Enum, 0, len(s.enums))
	for _, e := range s.enums {
		out = append(out, e)
	}
	return out
}

func (s *Set) Files() []File {
	if s == nil {
		return nil
	}
	out := make([]File, len(s.files))
	copy(out, s.files)
	return out
}

func skipPackage(pkg string) bool {
	switch {
	case pkg == "google.protobuf", strings.HasPrefix(pkg, "google.protobuf."):
		return true
	case pkg == "transformgen.options", strings.HasPrefix(pkg, "transformgen.options."):
		return true
	default:
		return false
	}
}

func collectMessage(
	out *Set,
	fileInfo *File,
	pkg string,
	goImport string,
	goPackage string,
	csharpNamespace string,
	sourceFile string,
	parent string,
	msg *descriptorpb.DescriptorProto,
) error {
	if msg.GetOptions().GetMapEntry() {
		return nil
	}
	protoName := msg.GetName()
	fullName := joinName(pkg, parent, protoName)
	for _, nested := range msg.GetEnumType() {
		e := Enum{
			FullName:     fullName + "." + nested.GetName(),
			ProtoPackage: pkg,
			ProtoName:    nested.GetName(),
			SourceFile:   sourceFile,
			Values:       enumValues(nested.GetName(), nested.GetValue()),
		}
		if _, ok := out.enums[e.FullName]; ok {
			return fmt.Errorf("%w: %s", ErrDuplicateMessage, e.FullName)
		}
		out.enums[e.FullName] = e
		fileInfo.Enums = append(fileInfo.Enums, e.FullName)
	}
	for _, nested := range msg.GetNestedType() {
		if err := collectMessage(out, fileInfo, pkg, goImport, goPackage, csharpNamespace, sourceFile, joinParent(parent, protoName), nested); err != nil {
			return err
		}
	}
	fields, oneofs, hasRetError, err := readFields(msg)
	if err != nil {
		return fmt.Errorf("%s: %w", fullName, err)
	}
	kind, _ := InferKindFromName(protoName)
	goInfo := GoMessage{ImportPath: goImport, PackageName: goPackage, TypeName: protoName}
	message := Message{
		Kind:          kind,
		FullName:      fullName,
		ProtoPackage:  pkg,
		ProtoName:     protoName,
		SourceFile:    sourceFile,
		Fields:        fields,
		Oneofs:        oneofs,
		HasRetError:   hasRetError,
		Go:            goInfo,
		CSharp:        CSharpMessage{Namespace: csharpNamespace, TypeName: protoName},
		GoImportPath:  goInfo.ImportPath,
		GoPackageName: goInfo.PackageName,
		GoTypeName:    goInfo.TypeName,
	}
	if existing, ok := out.messages[message.FullName]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicateMessage, existing.FullName)
	}
	out.messages[message.FullName] = message
	fileInfo.Messages = append(fileInfo.Messages, message.FullName)
	return nil
}

func readEnum(pkg string, sourceFile string, enum *descriptorpb.EnumDescriptorProto) (Enum, error) {
	name := enum.GetName()
	fullName := name
	if pkg != "" {
		fullName = pkg + "." + name
	}
	return Enum{
		FullName:     fullName,
		ProtoPackage: pkg,
		ProtoName:    name,
		SourceFile:   sourceFile,
		Values:       enumValues(name, enum.GetValue()),
	}, nil
}

func enumValues(enumName string, values []*descriptorpb.EnumValueDescriptorProto) []EnumValue {
	prefix := enumName + "_"
	out := make([]EnumValue, 0, len(values))
	for _, value := range values {
		valueName := value.GetName()
		if strings.HasPrefix(valueName, prefix) {
			valueName = strings.TrimPrefix(valueName, prefix)
		}
		if strings.Contains(valueName, "_") || valueName == strings.ToUpper(valueName) {
			valueName = pascalIdent(valueName)
		}
		out = append(out, EnumValue{Name: valueName, Number: value.GetNumber()})
	}
	return out
}

func readFields(msg *descriptorpb.DescriptorProto) ([]Field, []Oneof, bool, error) {
	fields := make([]Field, 0, len(msg.GetField()))
	hasRetError := false
	for _, field := range msg.GetField() {
		f := Field{
			Number:   field.GetNumber(),
			Name:     field.GetName(),
			JSONName: field.GetJsonName(),
		}
		if field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
			f.Cardinal = FieldRepeated
		} else {
			f.Cardinal = FieldSingular
		}
		switch field.GetType() {
		case descriptorpb.FieldDescriptorProto_TYPE_GROUP:
			typeName := strings.TrimPrefix(field.GetTypeName(), ".")
			f.TypeKind = FieldTypeMessage
			f.IsGroup = true
			f.FullType = typeName
			f.TypeName = shortTypeName(typeName)
		case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
			typeName := strings.TrimPrefix(field.GetTypeName(), ".")
			if entry, ok := mapEntryType(msg, field); ok {
				keyType, valueType, err := mapEntryCSharpTypes(entry)
				if err != nil {
					return nil, nil, false, fmt.Errorf("map field %s: %w", field.GetName(), err)
				}
				f.TypeKind = FieldTypeMap
				f.Cardinal = FieldSingular // Dictionary property, not List
				f.FullType = typeName
				f.TypeName = shortTypeName(typeName)
				f.MapKeyType = keyType
				f.MapValType = valueType
				fields = append(fields, f)
				continue
			}
			f.TypeKind = FieldTypeMessage
			f.FullType = typeName
			f.TypeName = shortTypeName(typeName)
		case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
			typeName := strings.TrimPrefix(field.GetTypeName(), ".")
			f.TypeKind = FieldTypeEnum
			f.FullType = typeName
			f.TypeName = shortTypeName(typeName)
			if field.GetName() == "ret" && shortTypeName(typeName) == "EMsgErrorType" {
				hasRetError = true
			}
		case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
			f.TypeKind = FieldTypeBytes
			f.ScalarType = "byte[]"
		default:
			scalar, ok := scalarCSharpType(field.GetType())
			if !ok {
				return nil, nil, false, fmt.Errorf("%w: type %v on %s", ErrUnsupportedField, field.GetType(), field.GetName())
			}
			f.TypeKind = FieldTypeScalar
			f.ScalarType = scalar
		}
		fields = append(fields, f)
	}
	oneofs, err := attachOneofs(msg, fields)
	if err != nil {
		return nil, nil, false, err
	}
	return fields, oneofs, hasRetError, nil
}

type oneofStub struct {
	name               string
	count32, count64   int
	count128, countRef int
	members            []int
}

func attachOneofs(msg *descriptorpb.DescriptorProto, fields []Field) ([]Oneof, error) {
	decls := msg.GetOneofDecl()
	if len(decls) == 0 {
		return nil, nil
	}
	stubs := make([]oneofStub, len(decls))
	for i, decl := range decls {
		stubs[i].name = decl.GetName()
	}
	fieldIndexByNumber := make(map[int32]int, len(fields))
	for i, f := range fields {
		fieldIndexByNumber[f.Number] = i
	}
	for _, pf := range msg.GetField() {
		if pf.OneofIndex == nil || pf.GetProto3Optional() {
			continue
		}
		idx := int(pf.GetOneofIndex())
		if idx < 0 || idx >= len(stubs) {
			return nil, fmt.Errorf("%w: oneof index %d on %s", ErrUnsupportedField, idx, pf.GetName())
		}
		fi, ok := fieldIndexByNumber[pf.GetNumber()]
		if !ok {
			continue
		}
		accountOneof(&stubs[idx], pf.GetType(), pf.GetTypeName())
		stubs[idx].members = append(stubs[idx].members, fi)
	}
	var oneofs []Oneof
	for _, s := range stubs {
		if len(s.members) == 0 {
			continue // synthetic / unused
		}
		unionType := oneofUnionType(s.count32, s.count64, s.count128, s.countRef)
		unionField := "__pbn__" + s.name
		oneof := Oneof{Name: s.name, UnionType: unionType}
		for i, fi := range s.members {
			pf := findProtoField(msg, fields[fi].Number)
			storage := oneofStorage(pf.GetType(), pf.GetTypeName())
			fields[fi].OneofName = s.name
			fields[fi].OneofUnionType = unionType
			fields[fi].OneofUnionField = unionField
			fields[fi].OneofStorage = storage
			fields[fi].IsFirstOneofMember = i == 0
			oneof.Fields = append(oneof.Fields, fields[fi].Name)
		}
		oneofs = append(oneofs, oneof)
	}
	return oneofs, nil
}

func findProtoField(msg *descriptorpb.DescriptorProto, number int32) *descriptorpb.FieldDescriptorProto {
	for _, field := range msg.GetField() {
		if field.GetNumber() == number {
			return field
		}
	}
	return &descriptorpb.FieldDescriptorProto{}
}

func accountOneof(s *oneofStub, t descriptorpb.FieldDescriptorProto_Type, typeName string) {
	switch t {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		s.count32++
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		s.count32++
		s.count64++
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		switch strings.TrimPrefix(typeName, ".") {
		case "google.protobuf.Timestamp", "google.protobuf.Duration":
			s.count64++
		default:
			s.countRef++
		}
	default:
		s.countRef++
	}
}

func oneofUnionType(count32, count64, count128, countRef int) string {
	if count128 != 0 {
		if countRef == 0 {
			return "DiscriminatedUnion128"
		}
		return "DiscriminatedUnion128Object"
	}
	if count64 != 0 {
		if countRef == 0 {
			return "DiscriminatedUnion64"
		}
		return "DiscriminatedUnion64Object"
	}
	if count32 != 0 {
		if countRef == 0 {
			return "DiscriminatedUnion32"
		}
		return "DiscriminatedUnion32Object"
	}
	return "DiscriminatedUnionObject"
}

func oneofStorage(t descriptorpb.FieldDescriptorProto_Type, typeName string) string {
	switch t {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "Boolean"
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		return "Int32"
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		return "Single"
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED32, descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		return "UInt32"
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "Double"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		return "Int64"
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED64, descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		return "UInt64"
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		switch strings.TrimPrefix(typeName, ".") {
		case "google.protobuf.Timestamp":
			return "DateTime"
		case "google.protobuf.Duration":
			return "TimeSpan"
		default:
			return "Object"
		}
	default:
		return "Object"
	}
}

func mapEntryType(msg *descriptorpb.DescriptorProto, field *descriptorpb.FieldDescriptorProto) (*descriptorpb.DescriptorProto, bool) {
	if field.GetLabel() != descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return nil, false
	}
	typeName := strings.TrimPrefix(field.GetTypeName(), ".")
	short := shortTypeName(typeName)
	for _, nested := range msg.GetNestedType() {
		if nested.GetName() == short && nested.GetOptions().GetMapEntry() {
			return nested, true
		}
	}
	return nil, false
}

func mapEntryCSharpTypes(entry *descriptorpb.DescriptorProto) (string, string, error) {
	var keyField, valueField *descriptorpb.FieldDescriptorProto
	for _, field := range entry.GetField() {
		switch field.GetName() {
		case "key":
			keyField = field
		case "value":
			valueField = field
		}
	}
	if keyField == nil || valueField == nil {
		return "", "", fmt.Errorf("%w: map entry missing key/value", ErrUnsupportedField)
	}
	keyType, err := fieldCSharpType(keyField)
	if err != nil {
		return "", "", fmt.Errorf("map key: %w", err)
	}
	valueType, err := fieldCSharpType(valueField)
	if err != nil {
		return "", "", fmt.Errorf("map value: %w", err)
	}
	return keyType, valueType, nil
}

func fieldCSharpType(field *descriptorpb.FieldDescriptorProto) (string, error) {
	switch field.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		return shortTypeName(strings.TrimPrefix(field.GetTypeName(), ".")), nil
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		return shortTypeName(strings.TrimPrefix(field.GetTypeName(), ".")), nil
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "byte[]", nil
	default:
		scalar, ok := scalarCSharpType(field.GetType())
		if !ok {
			return "", fmt.Errorf("%w: type %v", ErrUnsupportedField, field.GetType())
		}
		return scalar, nil
	}
}

func scalarCSharpType(t descriptorpb.FieldDescriptorProto_Type) (string, bool) {
	switch t {
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "double", true
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		return "float", true
	case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		return "long", true
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		return "ulong", true
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		return "int", true
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED32, descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		return "uint", true
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "bool", true
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string", true
	default:
		return "", false
	}
}

func shortTypeName(full string) string {
	if i := strings.LastIndex(full, "."); i >= 0 {
		return full[i+1:]
	}
	return full
}

func joinName(pkg, parent, name string) string {
	parts := make([]string, 0, 3)
	if pkg != "" {
		parts = append(parts, pkg)
	}
	if parent != "" {
		parts = append(parts, parent)
	}
	parts = append(parts, name)
	return strings.Join(parts, ".")
}

func joinParent(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

func pascalIdent(value string) string {
	if value == "" {
		return value
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-'
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(strings.ToLower(part[1:]))
		}
	}
	out := b.String()
	if out == "" {
		return value
	}
	return out
}

// InferKindFromName derives request/response/notify from the proto message name.
// Supports both suffix style (HeartbeatRequest) and TianLong middle-token style
// (MsgCtrReqLogin / MsgCtrResLogin / MsgCtrNtfLogin).
func InferKindFromName(name string) (MessageKind, bool) {
	switch {
	case strings.HasSuffix(name, "Request") || hasCamelToken(name, "Req"):
		return MessageKindRequest, true
	case strings.HasSuffix(name, "Response") || hasCamelToken(name, "Res"):
		return MessageKindResponse, true
	case strings.HasSuffix(name, "Notify") || hasCamelToken(name, "Ntf"):
		return MessageKindNotify, true
	default:
		return 0, false
	}
}

func hasCamelToken(name, token string) bool {
	for idx := 0; ; {
		rel := strings.Index(name[idx:], token)
		if rel < 0 {
			return false
		}
		i := idx + rel
		end := i + len(token)
		beforeOK := i == 0 || isLower(name[i-1])
		afterOK := end == len(name) || isUpper(name[end])
		if beforeOK && afterOK {
			return true
		}
		idx = i + 1
	}
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
func isLower(b byte) bool { return b >= 'a' && b <= 'z' }

func splitGoPackage(value string) (string, string) {
	if value == "" {
		return "", ""
	}
	parts := strings.Split(value, ";")
	if len(parts) == 1 {
		pathParts := strings.Split(parts[0], "/")
		return parts[0], pathParts[len(pathParts)-1]
	}
	return parts[0], parts[1]
}
