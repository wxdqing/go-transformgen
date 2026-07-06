package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/proto/options"
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

func TestRunGeneratesCSharpRequester(t *testing.T) {
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
	requester := readFile(t, filepath.Join(outDir, "PlayerRequester.cs"))
	if !strings.Contains(requester, "namespace Plan.Protocol;") || !strings.Contains(requester, "EncodeHeartbeatRequest") {
		t.Fatalf("PlayerRequester.cs missing requester content:\n%s", requester)
	}
	responder := readFile(t, filepath.Join(outDir, "PlayerResponder.cs"))
	if !strings.Contains(responder, "public interface IPlayerHandler") || !strings.Contains(responder, "DispatchRequest") {
		t.Fatalf("PlayerResponder.cs missing responder content:\n%s", responder)
	}
}

func writeDescriptorSet(t *testing.T, path string) {
	t.Helper()
	req := message("HeartbeatRequest", 1001, options.MessageKind_MESSAGE_KIND_REQUEST)
	resp := message("HeartbeatResponse", 1002, options.MessageKind_MESSAGE_KIND_RESPONSE)
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name:        proto.String("heartbeat.proto"),
		Package:     proto.String("transform"),
		Options:     &descriptorpb.FileOptions{GoPackage: proto.String("example.com/transform;transformpb"), CsharpNamespace: proto.String("Plan.Transform")},
		MessageType: []*descriptorpb.DescriptorProto{req, resp},
	}}}
	raw, err := proto.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func message(name string, id uint32, kind options.MessageKind) *descriptorpb.DescriptorProto {
	opts := &descriptorpb.MessageOptions{}
	proto.SetExtension(opts, options.E_MessageId, id)
	proto.SetExtension(opts, options.E_MessageKind, kind)
	return &descriptorpb.DescriptorProto{Name: proto.String(name), Options: opts}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
