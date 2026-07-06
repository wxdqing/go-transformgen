package descriptor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wxdqing/go-transformgen/proto/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestLoadExtractsMessageOptionsAndGoTypes(t *testing.T) {
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
					message("HeartbeatRequest", 1001, options.MessageKind_MESSAGE_KIND_REQUEST),
					message("HeartbeatResponse", 1002, options.MessageKind_MESSAGE_KIND_RESPONSE),
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
	if req.ID != 1001 || req.Kind != MessageKindRequest || req.FullName != "transform.HeartbeatRequest" || req.ProtoPackage != "transform" || req.ProtoName != "HeartbeatRequest" {
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
	if resp.ID != 1002 || resp.Kind != MessageKindResponse {
		t.Fatalf("response = %+v", resp)
	}
}

func TestLoadRejectsMessageIDOutsideFileRange(t *testing.T) {
	fileOpts := &descriptorpb.FileOptions{
		GoPackage: proto.String("resource/protocol/src/transform;transformpb"),
	}
	proto.SetExtension(fileOpts, options.E_MessageIdMin, uint32(1000))
	proto.SetExtension(fileOpts, options.E_MessageIdMax, uint32(1999))
	set := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:        proto.String("heartbeat.proto"),
				Package:     proto.String("transform"),
				Options:     fileOpts,
				MessageType: []*descriptorpb.DescriptorProto{message("BattleFinishedNotify", 2001, options.MessageKind_MESSAGE_KIND_NOTIFY)},
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

	if _, err := Load(path); !errors.Is(err, ErrMessageIDOutOfRange) {
		t.Fatalf("Load() error = %v, want ErrMessageIDOutOfRange", err)
	}
}

func message(name string, id uint32, kind options.MessageKind) *descriptorpb.DescriptorProto {
	opts := &descriptorpb.MessageOptions{}
	proto.SetExtension(opts, options.E_MessageId, id)
	proto.SetExtension(opts, options.E_MessageKind, kind)
	return &descriptorpb.DescriptorProto{
		Name:    proto.String(name),
		Options: opts,
	}
}
