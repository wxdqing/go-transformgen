package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestRunRequiresCoreArguments(t *testing.T) {
	err := run([]string{"--target", "go"})
	if !errors.Is(err, errMissingArgument) {
		t.Fatalf("run() error = %v, want errMissingArgument", err)
	}
}

func TestRunRejectsTemplateDirUntilSupported(t *testing.T) {
	err := run([]string{"--template-dir", "templates"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined: -template-dir") {
		t.Fatalf("run() error = %v, want undefined template-dir flag", err)
	}
}

func TestParseGoImportFlag(t *testing.T) {
	imports := importMap{}
	if err := imports.Set("registry=example.com/registry"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := imports["registry"]; got != "example.com/registry" {
		t.Fatalf("registry import = %q", got)
	}
}

func TestRunPassesGoImportAndRuntimeOptions(t *testing.T) {
	dir := t.TempDir()
	protoSet := filepath.Join(dir, "transform.pbset")
	definesDir := filepath.Join(dir, "defines")
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(definesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDescriptorSet(t, protoSet)
	if err := os.WriteFile(filepath.Join(definesDir, "player.yaml"), []byte(`version: 1
model_name: player
ctx_import: context
rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: context.Context
`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"--proto-set", protoSet,
		"--defines-dir", definesDir,
		"--target", "go",
		"--side", "requester,responder",
		"--runtime", "emit",
		"--go-import", "proto=example.com/protobuf/proto",
		"--out", outDir,
		"--package", "protocolpb",
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	module := readFile(t, filepath.Join(outDir, "player.go"))
	runtimeFrame := readFile(t, filepath.Join(outDir, "runtime_frame.go"))
	if !strings.Contains(module, `"example.com/protobuf/proto"`) {
		t.Fatalf("player.go missing proto override:\n%s", module)
	}
	if strings.Contains(module, "github.com/wxdqing/go-transformgen/runtime") {
		t.Fatalf("player.go should not import transformgen runtime in emit mode:\n%s", module)
	}
	if !strings.Contains(runtimeFrame, "type FrameCodec interface") {
		t.Fatalf("runtime_frame.go missing emitted runtime:\n%s", runtimeFrame)
	}
}

func TestRunWritesMsgidLockAndKeepsIdsStable(t *testing.T) {
	dir := t.TempDir()
	protoSet := filepath.Join(dir, "transform.pbset")
	definesDir := filepath.Join(dir, "defines")
	outDir := filepath.Join(dir, "out")
	lockPath := filepath.Join(dir, "msgid.lock.yaml")
	if err := os.MkdirAll(definesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDescriptorSet(t, protoSet)
	if err := os.WriteFile(filepath.Join(definesDir, "player.yaml"), []byte(`version: 1
model_name: player
rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: context.Context
`), 0o644); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--proto-set", protoSet,
		"--defines-dir", definesDir,
		"--msgid-lock", lockPath,
		"--target", "go",
		"--side", "requester,responder",
		"--runtime", "emit",
		"--out", outDir,
		"--package", "protocolpb",
	}
	if err := run(args); err != nil {
		t.Fatalf("first run: %v", err)
	}
	first := readFile(t, lockPath)
	if !strings.Contains(first, "HeartbeatRequest:") || !strings.Contains(first, "HeartbeatResponse:") {
		t.Fatalf("lock missing messages:\n%s", first)
	}
	// Pin an artificial id and ensure a second run keeps it.
	if err := os.WriteFile(lockPath, []byte("messages:\n  HeartbeatRequest: 200000001\n  HeartbeatResponse: 100000001\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(args); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second := readFile(t, lockPath)
	if !strings.Contains(second, "HeartbeatRequest: 200000001") || !strings.Contains(second, "HeartbeatResponse: 100000001") {
		t.Fatalf("lock ids not preserved:\n%s", second)
	}
	messages := readFile(t, filepath.Join(outDir, "protocol_messages.go"))
	if !strings.Contains(messages, "200000001") || !strings.Contains(messages, "100000001") {
		t.Fatalf("generated code missing locked ids:\n%s", messages)
	}
}

func TestRunGeneratesCSharpMessageTypes(t *testing.T) {
	dir := t.TempDir()
	protoSet := filepath.Join(dir, "transform.pbset")
	definesDir := filepath.Join(dir, "defines")
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(definesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDescriptorSet(t, protoSet)
	if err := os.WriteFile(filepath.Join(definesDir, "player.yaml"), []byte(`version: 1
model_name: player
rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: context.Context
`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"--proto-set", protoSet,
		"--defines-dir", definesDir,
		"--target", "csharp",
		"--side", "requester,responder",
		"--runtime", "emit",
		"--out", outDir,
		"--package", "Plan.Protocol",
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	server := readFile(t, filepath.Join(outDir, "EMsgToServerType.cs"))
	if !strings.Contains(server, "public enum EMsgToServerType") || !strings.Contains(server, "HeartbeatRequest = 2") {
		t.Fatalf("EMsgToServerType.cs missing request enum:\n%s", server)
	}
	client := readFile(t, filepath.Join(outDir, "EMsgToClientType.cs"))
	if !strings.Contains(client, "public enum EMsgToClientType") || !strings.Contains(client, "HeartbeatResponse = 1") {
		t.Fatalf("EMsgToClientType.cs missing response enum:\n%s", client)
	}
	binding := readFile(t, filepath.Join(outDir, "EMsgType.cs"))
	for _, want := range []string{
		"public interface IProtoBufToServer : IProtoBuf<EMsgToServerType> { }",
		"public partial class HeartbeatRequest : IProtoBufToServer",
		"public const EMsgToServerType MsgType = EMsgToServerType.HeartbeatRequest;",
		"public partial class HeartbeatResponse : IProtoBufToClient",
	} {
		if !strings.Contains(binding, want) {
			t.Fatalf("EMsgType.cs missing %q:\n%s", want, binding)
		}
	}
	messages := readFile(t, filepath.Join(outDir, "Heartbeat.cs"))
	for _, want := range []string{
		"public partial class HeartbeatRequest : global::ProtoBuf.IExtensible",
		"public long ClientTime { get; set; }",
		"public ulong Sequence { get; set; }",
	} {
		if !strings.Contains(messages, want) {
			t.Fatalf("Heartbeat.cs missing %q:\n%s", want, messages)
		}
	}
}

// TestRunGeneratesCppProtocol verifies the CLI wires the C++ target end to end.
func TestRunGeneratesCppProtocol(t *testing.T) {
	// Prepare a minimal descriptor set and one module definition.
	dir := t.TempDir()
	protoSet := filepath.Join(dir, "transform.pbset")
	definesDir := filepath.Join(dir, "defines")
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(definesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDescriptorSet(t, protoSet)
	if err := os.WriteFile(filepath.Join(definesDir, "player.yaml"), []byte(`version: 1
model_name: player
rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: context.Context
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Generate with the C++ target and an include prefix.
	err := run([]string{
		"--proto-set", protoSet,
		"--defines-dir", definesDir,
		"--target", "cpp",
		"--side", "requester,responder",
		"--runtime", "emit",
		"--out", outDir,
		"--package", "transform",
		"--cpp-proto-include-prefix", "protos/pb",
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	// Check enums, includes, and module classes in the emitted files.
	header := readFile(t, filepath.Join(outDir, "protocol_messages.hpp"))
	for _, want := range []string{
		"namespace transform {",
		"enum class EMsgToServerType : std::uint32_t {",
		"enum class EMsgToClientType : std::uint32_t {",
		"#include \"protos/pb/heartbeat.pb.h\"",
	} {
		if !strings.Contains(header, want) {
			t.Fatalf("protocol_messages.hpp missing %q:\n%s", want, header)
		}
	}
	requester := readFile(t, filepath.Join(outDir, "player_requester.hpp"))
	if !strings.Contains(requester, "namespace transform::player {") || !strings.Contains(requester, "class Requester {") {
		t.Fatalf("player_requester.hpp missing module namespace or class:\n%s", requester)
	}
	responder := readFile(t, filepath.Join(outDir, "player_responder.hpp"))
	if !strings.Contains(responder, "class ResponderHandler {") {
		t.Fatalf("player_responder.hpp missing handler interface:\n%s", responder)
	}
}

func writeDescriptorSet(t *testing.T, path string) {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:    proto.String("heartbeat.proto"),
		Package: proto.String("transform"),
		Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/transform;transformpb"), CsharpNamespace: proto.String("Plan.Transform")},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("HeartbeatRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("client_time"), JsonName: proto.String("clientTime"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
					{Name: proto.String("sequence"), JsonName: proto.String("sequence"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum()},
				},
			},
			{Name: proto.String("HeartbeatResponse")},
		},
	}}}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
