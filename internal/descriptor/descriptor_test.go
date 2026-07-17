package descriptor

import (
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestLoadExtractsMessageAndGoTypes(t *testing.T) {
	set := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("heartbeat.proto"),
				Package: proto.String("transform"),
				Options: &descriptorpb.FileOptions{
					GoPackage:       proto.String("resource/protocol/src/transform;transformpb"),
					CsharpNamespace: proto.String("Plan.Transform"),
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{Name: proto.String("HeartbeatRequest")},
					{Name: proto.String("HeartbeatResponse")},
				},
			},
		},
	}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transform.pbset")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	req, ok := loaded.Message("transform.HeartbeatRequest")
	if !ok {
		t.Fatal("request message not found")
	}
	if req.Kind != MessageKindRequest || req.FullName != "transform.HeartbeatRequest" || req.ProtoPackage != "transform" || req.ProtoName != "HeartbeatRequest" {
		t.Fatalf("request language-neutral fields = %+v", req)
	}
	if req.Go.ImportPath != "resource/protocol/src/transform" || req.Go.PackageName != "transformpb" || req.Go.TypeName != "HeartbeatRequest" {
		t.Fatalf("request Go = %+v", req)
	}
	if req.CSharp.Namespace != "Plan.Transform" || req.CSharp.TypeName != "HeartbeatRequest" {
		t.Fatalf("request = %+v", req)
	}
	resp, ok := loaded.Message("transform.HeartbeatResponse")
	if !ok {
		t.Fatal("response message not found")
	}
	if resp.Kind != MessageKindResponse {
		t.Fatalf("response = %+v", resp)
	}
}

func TestInferKindFromName(t *testing.T) {
	cases := []struct {
		name string
		kind MessageKind
		ok   bool
	}{
		{"HeartbeatRequest", MessageKindRequest, true},
		{"HeartbeatResponse", MessageKindResponse, true},
		{"BattleFinishedNotify", MessageKindNotify, true},
		{"MsgCtrReqLogin", MessageKindRequest, true},
		{"MsgCtrResLogin", MessageKindResponse, true},
		{"MsgCtrNtfLogin", MessageKindNotify, true},
		{"MsgDataAccount", 0, false},
	}
	for _, tc := range cases {
		got, ok := InferKindFromName(tc.name)
		if ok != tc.ok || got != tc.kind {
			t.Fatalf("InferKindFromName(%q) = (%d,%v), want (%d,%v)", tc.name, got, ok, tc.kind, tc.ok)
		}
	}
}

func TestLoadMapAndGroupFields(t *testing.T) {
	mapEntry := &descriptorpb.DescriptorProto{
		Name:    proto.String("MentionCountsEntry"),
		Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
		Field: []*descriptorpb.FieldDescriptorProto{
			{Name: proto.String("key"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
			{Name: proto.String("value"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
		},
	}
	set := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("chat.proto"),
			Package: proto.String("transform"),
			Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/transform;transformpb")},
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("ChatMessageNotify"),
					NestedType: []*descriptorpb.DescriptorProto{
						mapEntry,
						{Name: proto.String("ExtraGroup"), Field: []*descriptorpb.FieldDescriptorProto{
							{Name: proto.String("note"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
						}},
					},
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("mention_counts"),
							JsonName: proto.String("mentionCounts"),
							Number:   proto.Int32(5),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".transform.ChatMessageNotify.MentionCountsEntry"),
						},
						{
							Name:     proto.String("extra"),
							JsonName: proto.String("extra"),
							Number:   proto.Int32(6),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_GROUP.Enum(),
							TypeName: proto.String(".transform.ChatMessageNotify.ExtraGroup"),
						},
					},
				},
			},
		}},
	}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transform.pbset")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	msg, ok := loaded.Message("transform.ChatMessageNotify")
	if !ok {
		t.Fatal("ChatMessageNotify missing")
	}
	if len(msg.Fields) != 2 {
		t.Fatalf("fields = %+v, want 2", msg.Fields)
	}
	mapField := msg.Fields[0]
	if mapField.TypeKind != FieldTypeMap || mapField.MapKeyType != "string" || mapField.MapValType != "int" {
		t.Fatalf("map field = %+v", mapField)
	}
	groupField := msg.Fields[1]
	if !groupField.IsGroup || groupField.TypeKind != FieldTypeMessage || groupField.TypeName != "ExtraGroup" {
		t.Fatalf("group field = %+v", groupField)
	}
	if _, ok := loaded.Message("transform.ChatMessageNotify.MentionCountsEntry"); ok {
		t.Fatal("map entry message should not be emitted")
	}
	if _, ok := loaded.Message("transform.ChatMessageNotify.ExtraGroup"); !ok {
		t.Fatal("group message type should be collected")
	}
}

func TestLoadOneofFields(t *testing.T) {
	set := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("chat.proto"),
			Package: proto.String("transform"),
			Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/transform;transformpb")},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("ChatMessageNotify"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: proto.String("attachment")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name: proto.String("text_attachment"), JsonName: proto.String("textAttachment"), Number: proto.Int32(6),
						Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						OneofIndex: proto.Int32(0),
					},
					{
						Name: proto.String("tag_attachment"), JsonName: proto.String("tagAttachment"), Number: proto.Int32(7),
						Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".transform.MsgDataTag"), OneofIndex: proto.Int32(0),
					},
				},
			}},
		}},
	}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transform.pbset")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	msg, ok := loaded.Message("transform.ChatMessageNotify")
	if !ok {
		t.Fatal("message missing")
	}
	if len(msg.Oneofs) != 1 || msg.Oneofs[0].Name != "attachment" {
		t.Fatalf("oneofs = %+v", msg.Oneofs)
	}
	if msg.Oneofs[0].UnionType != "DiscriminatedUnionObject" {
		t.Fatalf("union type = %q", msg.Oneofs[0].UnionType)
	}
	if !msg.Fields[0].IsFirstOneofMember || msg.Fields[0].OneofUnionField != "__pbn__attachment" {
		t.Fatalf("first oneof field = %+v", msg.Fields[0])
	}
	if msg.Fields[1].IsFirstOneofMember || msg.Fields[1].OneofName != "attachment" {
		t.Fatalf("second oneof field = %+v", msg.Fields[1])
	}
}

func TestLoadSkipsGoogleProtobufPackage(t *testing.T) {
	set := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:        proto.String("descriptor.proto"),
				Package:     proto.String("google.protobuf"),
				MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("FileDescriptorSet")}},
			},
			{
				Name:        proto.String("heartbeat.proto"),
				Package:     proto.String("transform"),
				Options:     &descriptorpb.FileOptions{GoPackage: proto.String("example.com/transform;transformpb")},
				MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("HeartbeatRequest")}},
			},
		},
	}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transform.pbset")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, ok := loaded.Message("google.protobuf.FileDescriptorSet"); ok {
		t.Fatal("google.protobuf messages should be skipped")
	}
	if _, ok := loaded.Message("transform.HeartbeatRequest"); !ok {
		t.Fatal("business message missing")
	}
}
