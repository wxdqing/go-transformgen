package csharptarget

import (
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

func TestRenderEmitsDirectionEnumsAndBindings(t *testing.T) {
	files, err := Render(testModel(), testDescriptor(), Options{Runtime: RuntimeModeEmit})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(files)
	for _, path := range []string{"EMsgToServerType.cs", "EMsgToClientType.cs", "EMsgType.cs", "Heartbeat.cs"} {
		if byPath[path] == "" {
			t.Fatalf("missing %s: %+v", path, files)
		}
	}
	server := byPath["EMsgToServerType.cs"]
	if !strings.Contains(server, "HeartbeatRequest = 200012345,") {
		t.Fatalf("EMsgToServerType.cs missing request:\n%s", server)
	}
	binding := byPath["EMsgType.cs"]
	for _, want := range []string{
		"public interface IRetErrorType",
		"public partial class HeartbeatResponse : IProtoBufToClient, IRetErrorType",
		"public partial class HeartbeatRequest : IProtoBufToServer",
	} {
		if !strings.Contains(binding, want) {
			t.Fatalf("EMsgType.cs missing %q:\n%s", want, binding)
		}
	}
}

func TestRenderEmitsProtobufNetMessages(t *testing.T) {
	files, err := Render(testModel(), testDescriptor(), Options{Runtime: RuntimeModeEmit})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	heartbeat := filesByPath(files)["Heartbeat.cs"]
	for _, want := range []string{
		"Input: heartbeat.proto",
		"[global::ProtoBuf.ProtoContract()]",
		"public partial class HeartbeatRequest : global::ProtoBuf.IExtensible",
		"[global::ProtoBuf.ProtoMember(1)]",
		"public long ClientTime { get; set; }",
		"public ulong Sequence { get; set; }",
		"public EMsgErrorType Ret { get; set; }",
	} {
		if !strings.Contains(heartbeat, want) {
			t.Fatalf("Heartbeat.cs missing %q:\n%s", want, heartbeat)
		}
	}
	common := filesByPath(files)["Common.cs"]
	for _, want := range []string{
		"public enum EMsgErrorType",
		"None = 0,",
		"InvalidArgument = 1,",
		"public partial class MsgDataTag : global::ProtoBuf.IExtensible",
		"[global::System.ComponentModel.DefaultValue(\"\")]",
		"public string Name { get; set; } = \"\";",
	} {
		if !strings.Contains(common, want) {
			t.Fatalf("Common.cs missing %q:\n%s", want, common)
		}
	}
}

func TestRenderOneofFields(t *testing.T) {
	desc := descriptor.NewSetWithFiles(
		[]descriptor.File{{
			Name:     "chat.proto",
			BaseName: "chat.proto",
			Messages: []string{"transform.ChatMessageNotify"},
		}},
		nil,
		[]descriptor.Message{{
			FullName:  "transform.ChatMessageNotify",
			ProtoName: "ChatMessageNotify",
			Oneofs: []descriptor.Oneof{{
				Name: "attachment", UnionType: "DiscriminatedUnionObject",
				Fields: []string{"text_attachment", "tag_attachment"},
			}},
			Fields: []descriptor.Field{
				{
					Number: 6, Name: "text_attachment", JSONName: "textAttachment",
					TypeKind: descriptor.FieldTypeScalar, ScalarType: "string",
					OneofName: "attachment", OneofUnionType: "DiscriminatedUnionObject",
					OneofUnionField: "__pbn__attachment", OneofStorage: "Object", IsFirstOneofMember: true,
				},
				{
					Number: 7, Name: "tag_attachment", JSONName: "tagAttachment",
					TypeKind: descriptor.FieldTypeMessage, TypeName: "MsgDataTag",
					OneofName: "attachment", OneofUnionType: "DiscriminatedUnionObject",
					OneofUnionField: "__pbn__attachment", OneofStorage: "Object",
				},
			},
		}},
	)
	files, err := Render(&model.Model{}, desc, Options{Runtime: RuntimeModeEmit})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	chat := filesByPath(files)["Chat.cs"]
	for _, want := range []string{
		"public string TextAttachment",
		"get => __pbn__attachment.Is(6) ? ((string)__pbn__attachment.Object) : default;",
		"set => __pbn__attachment = new global::ProtoBuf.DiscriminatedUnionObject(6, value);",
		"public bool ShouldSerializeTextAttachment() => __pbn__attachment.Is(6);",
		"private global::ProtoBuf.DiscriminatedUnionObject __pbn__attachment;",
		"public MsgDataTag TagAttachment",
		"public AttachmentOneofCase AttachmentCase => (AttachmentOneofCase)__pbn__attachment.Discriminator;",
		"public enum AttachmentOneofCase",
		"TextAttachment = 6,",
		"TagAttachment = 7,",
	} {
		if !strings.Contains(chat, want) {
			t.Fatalf("Chat.cs missing %q:\n%s", want, chat)
		}
	}
}

func TestRenderMapAndGroupFields(t *testing.T) {
	desc := descriptor.NewSetWithFiles(
		[]descriptor.File{{
			Name:     "chat.proto",
			BaseName: "chat.proto",
			Messages: []string{"transform.ChatMessageNotify", "transform.ChatMessageNotify.ExtraGroup"},
		}},
		nil,
		[]descriptor.Message{
			{
				FullName:  "transform.ChatMessageNotify",
				ProtoName: "ChatMessageNotify",
				Fields: []descriptor.Field{
					{
						Number: 5, Name: "mention_counts", JSONName: "mentionCounts",
						TypeKind: descriptor.FieldTypeMap, MapKeyType: "string", MapValType: "int",
					},
					{
						Number: 6, Name: "extra", JSONName: "extra",
						TypeKind: descriptor.FieldTypeMessage, TypeName: "ExtraGroup", IsGroup: true,
					},
				},
			},
			{
				FullName:  "transform.ChatMessageNotify.ExtraGroup",
				ProtoName: "ExtraGroup",
				Fields: []descriptor.Field{
					{Number: 1, Name: "note", JSONName: "note", TypeKind: descriptor.FieldTypeScalar, ScalarType: "string", Cardinal: descriptor.FieldSingular},
				},
			},
		},
	)
	files, err := Render(&model.Model{}, desc, Options{Runtime: RuntimeModeEmit})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	chat := filesByPath(files)["Chat.cs"]
	for _, want := range []string{
		"[global::ProtoBuf.ProtoMap]",
		"public global::System.Collections.Generic.Dictionary<string, int> MentionCounts { get; } = new global::System.Collections.Generic.Dictionary<string, int>();",
		"[global::ProtoBuf.ProtoMember(6, DataFormat = global::ProtoBuf.DataFormat.Group)]",
		"public ExtraGroup Extra { get; set; }",
		"public partial class ExtraGroup : global::ProtoBuf.IExtensible",
	} {
		if !strings.Contains(chat, want) {
			t.Fatalf("Chat.cs missing %q:\n%s", want, chat)
		}
	}
}

func TestRenderCSharpRejectsImportRuntime(t *testing.T) {
	_, err := Render(testModel(), testDescriptor(), Options{Runtime: RuntimeModeImport})
	if err == nil {
		t.Fatal("Render() error = nil, want import runtime rejection")
	}
}

func TestRenderTemplateReturnsParseError(t *testing.T) {
	_, err := renderTemplate("broken.cs", "{{", nil)
	if err == nil {
		t.Fatal("renderTemplate() error = nil, want parse error")
	}
}

func filesByPath(files []File) map[string]string {
	out := make(map[string]string, len(files))
	for _, file := range files {
		out[file.Path] = string(file.Content)
	}
	return out
}

func testModel() *model.Model {
	return &model.Model{Modules: []model.Module{{
		Name:          "player",
		ConstName:     "ModelNamePlayer",
		InterfaceName: "Player",
		RPCs: []model.RPC{{
			Method: "Login",
			Request: descriptor.Message{
				ID:          200012345,
				Kind:        descriptor.MessageKindRequest,
				FullName:    "transform.HeartbeatRequest",
				ProtoName:   "HeartbeatRequest",
				HasRetError: false,
				CSharp:      descriptor.CSharpMessage{TypeName: "HeartbeatRequest"},
			},
			Response: descriptor.Message{
				ID:          100067890,
				Kind:        descriptor.MessageKindResponse,
				FullName:    "transform.HeartbeatResponse",
				ProtoName:   "HeartbeatResponse",
				HasRetError: true,
				CSharp:      descriptor.CSharpMessage{TypeName: "HeartbeatResponse"},
			},
		}},
	}}}
}

func testDescriptor() *descriptor.Set {
	return descriptor.NewSetWithFiles(
		[]descriptor.File{{
			Name:     "example/transform/common.proto",
			BaseName: "common.proto",
			Enums:    []string{"transform.example.EMsgErrorType"},
			Messages: []string{"transform.example.MsgDataTag"},
		}, {
			Name:     "example/transform/heartbeat.proto",
			BaseName: "heartbeat.proto",
			Messages: []string{"transform.example.HeartbeatRequest", "transform.example.HeartbeatResponse"},
		}},
		[]descriptor.Enum{{
			FullName:  "transform.example.EMsgErrorType",
			ProtoName: "EMsgErrorType",
			Values: []descriptor.EnumValue{
				{Name: "None", Number: 0},
				{Name: "InvalidArgument", Number: 1},
			},
		}},
		[]descriptor.Message{
			{
				FullName:  "transform.example.MsgDataTag",
				ProtoName: "MsgDataTag",
				Fields: []descriptor.Field{
					{Number: 1, Name: "name", JSONName: "name", Cardinal: descriptor.FieldSingular, TypeKind: descriptor.FieldTypeScalar, ScalarType: "string"},
					{Number: 2, Name: "weight", JSONName: "weight", Cardinal: descriptor.FieldSingular, TypeKind: descriptor.FieldTypeScalar, ScalarType: "int"},
				},
			},
			{
				FullName:  "transform.example.HeartbeatRequest",
				ProtoName: "HeartbeatRequest",
				Fields: []descriptor.Field{
					{Number: 1, Name: "client_time", JSONName: "clientTime", Cardinal: descriptor.FieldSingular, TypeKind: descriptor.FieldTypeScalar, ScalarType: "long"},
					{Number: 2, Name: "sequence", JSONName: "sequence", Cardinal: descriptor.FieldSingular, TypeKind: descriptor.FieldTypeScalar, ScalarType: "ulong"},
				},
			},
			{
				FullName:    "transform.example.HeartbeatResponse",
				ProtoName:   "HeartbeatResponse",
				HasRetError: true,
				Fields: []descriptor.Field{
					{Number: 1, Name: "ret", JSONName: "ret", Cardinal: descriptor.FieldSingular, TypeKind: descriptor.FieldTypeEnum, TypeName: "EMsgErrorType", FullType: "transform.example.EMsgErrorType"},
					{Number: 2, Name: "server_time", JSONName: "serverTime", Cardinal: descriptor.FieldSingular, TypeKind: descriptor.FieldTypeScalar, ScalarType: "long"},
				},
			},
		},
	)
}
