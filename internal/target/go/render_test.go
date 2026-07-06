package gotarget

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

func TestRenderProducesGoRegistrationFile(t *testing.T) {
	m := &model.Model{Modules: []model.Module{{
		Name:          "player",
		ConstName:     "ModelNamePlayer",
		InterfaceName: "Player",
		RPCs: []model.RPC{{
			Method:        "Heartbeat",
			Ctx:           "grainactor.Context",
			CtxImportPath: "apps/common/runtime/stateful/grainactor",
			Request: descriptor.Message{
				ID:            1001,
				Kind:          descriptor.MessageKindRequest,
				FullName:      "transform.HeartbeatRequest",
				GoImportPath:  "github.com/wxdqing/go-transformgen/example/transform",
				GoPackageName: "transformpb",
				GoTypeName:    "HeartbeatRequest",
			},
			Response: descriptor.Message{
				ID:            1002,
				Kind:          descriptor.MessageKindResponse,
				FullName:      "transform.HeartbeatResponse",
				GoImportPath:  "github.com/wxdqing/go-transformgen/example/transform",
				GoPackageName: "transformpb",
				GoTypeName:    "HeartbeatResponse",
			},
		}},
		Notifies: []model.Notify{{
			Method:        "BattleFinished",
			Ctx:           "grainactor.Context",
			CtxImportPath: "apps/common/runtime/stateful/grainactor",
			Message: descriptor.Message{
				ID:            2001,
				Kind:          descriptor.MessageKindNotify,
				FullName:      "transform.BattleFinishedNotify",
				GoImportPath:  "github.com/wxdqing/go-transformgen/example/transform",
				GoPackageName: "transformpb",
				GoTypeName:    "BattleFinishedNotify",
			},
		}},
	}}}

	files, err := Render(m, Options{
		Package: "protocolpb",
		Sides:   []string{"requester", "responder"},
		Runtime: RuntimeModeImport,
		Imports: ImportPaths{
			Frame:    "example.com/runtime/frame",
			Registry: "example.com/runtime/registry",
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("len(files) = %d, want 3", len(files))
	}
	byPath := filesByPath(files)
	protocol := string(byPath["protocol.go"].Content)
	messages := string(byPath["protocol_messages.go"].Content)
	module := string(byPath["player.go"].Content)
	if protocol == "" {
		t.Fatalf("missing protocol.go: %+v", files)
	}
	if messages == "" {
		t.Fatalf("missing protocol_messages.go: %+v", files)
	}
	if module == "" {
		t.Fatalf("missing player.go: %+v", files)
	}
	for _, want := range []string{
		"package protocolpb",
		`bootstrap "gitee.com/wxdqing/fx-bootstrap"`,
		`"go.uber.org/fx"`,
		"type Module struct",
		"type Protocol = Module",
		"type HandlerModule interface",
		"ModuleName() string",
		"Module() any",
		"type HandlerModuleOut struct",
		`Module HandlerModule ` + "`group:\"transformgen_handler_modules\"`",
		"type HandlerModuleWithBean[T HandlerModule] struct",
		"func NewHandlerModule(module HandlerModule) HandlerModuleOut",
		"func NewHandlerModuleWithBean[T HandlerModule](module T) HandlerModuleWithBean[T]",
		"type moduleParams struct",
		`Modules []HandlerModule ` + "`group:\"transformgen_handler_modules\"`",
		"type Provider struct",
		"var _ bootstrap.Provider = Provider{}",
		"func (p Provider) Register() any",
		"func (Provider) OnStart() any",
		"func NewModule(codec frame.FrameCodec, p moduleParams) *Module",
		"func (m *Module) Start(_ context.Context) error",
		"for _, module := range m.modules",
		"m.RegisterHandlers(module.ModuleName(), module.Module())",
		"func NewProtocol",
		"func PackMessage",
		"func RegisterHandlers",
		"case ModelNamePlayer:",
		"registerPlayerHandlers",
	} {
		if !strings.Contains(protocol, want) {
			t.Fatalf("protocol file missing %q:\n%s", want, protocol)
		}
	}
	for _, want := range []string{
		"package protocolpb",
		`transformpb "github.com/wxdqing/go-transformgen/example/transform"`,
		"const MessageIDHeartbeatRequest uint32 = 1001",
		"RegisterMessages",
		"return &transformpb.HeartbeatRequest{}",
	} {
		if !strings.Contains(messages, want) {
			t.Fatalf("messages file missing %q:\n%s", want, messages)
		}
	}
	for _, want := range []string{
		"package protocolpb",
		`"apps/common/runtime/stateful/grainactor"`,
		`transformpb "github.com/wxdqing/go-transformgen/example/transform"`,
		"const ModelNamePlayer = \"player\"",
		"type Player interface",
		"Heartbeat(ctx grainactor.Context, req *transformpb.HeartbeatRequest) (*transformpb.HeartbeatResponse, error)",
		"BattleFinished(ctx grainactor.Context, msg *transformpb.BattleFinishedNotify) error",
		"func registerPlayerHandlers",
		"EncodeHeartbeatRequest",
		"codec.EncodeFrame(frame.Head{MessageID: MessageIDHeartbeatRequest, RequestID: requestID}, body)",
		"DecodeHeartbeatResponse",
		"EncodeBattleFinishedNotify",
	} {
		if !strings.Contains(module, want) {
			t.Fatalf("module file missing %q:\n%s", want, module)
		}
	}
	for _, unwanted := range []string{
		"type PlayerNotify interface",
		"RegisterPlayerNotifyHandlers",
		"func RegisterPlayerHandlers",
	} {
		if strings.Contains(module, unwanted) {
			t.Fatalf("module file contains %q:\n%s", unwanted, module)
		}
	}
}

func filesByPath(files []File) map[string]File {
	out := make(map[string]File, len(files))
	for _, file := range files {
		out[file.Path] = file
	}
	return out
}

func TestGoTemplateFileContainsRequesterAndNotifySections(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("templates", "module.go.tmpl"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Encode{{ .Method }}Request",
		"Decode{{ .Method }}Response",
		"Encode{{ .Method }}Notify",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("template missing %q", want)
		}
	}
}

func TestRenderHonorsSideSelection(t *testing.T) {
	m := testModel()

	requesterFiles, err := Render(m, Options{Package: "protocolpb", Sides: []string{"requester"}})
	if err != nil {
		t.Fatalf("Render(requester) error = %v", err)
	}
	requesterModule := string(filesByPath(requesterFiles)["player.go"].Content)
	if !strings.Contains(requesterModule, "func EncodeHeartbeatRequest") {
		t.Fatalf("requester module missing EncodeHeartbeatRequest:\n%s", requesterModule)
	}
	if strings.Contains(requesterModule, "type Player interface") || strings.Contains(requesterModule, "func registerPlayerHandlers") {
		t.Fatalf("requester module contains responder code:\n%s", requesterModule)
	}
	if _, ok := filesByPath(requesterFiles)["protocol.go"]; ok {
		t.Fatalf("requester output should not include protocol.go: %+v", requesterFiles)
	}

	responderFiles, err := Render(m, Options{Package: "protocolpb", Sides: []string{"responder"}})
	if err != nil {
		t.Fatalf("Render(responder) error = %v", err)
	}
	responderModule := string(filesByPath(responderFiles)["player.go"].Content)
	if !strings.Contains(responderModule, "type Player interface") || !strings.Contains(responderModule, "func registerPlayerHandlers") {
		t.Fatalf("responder module missing responder code:\n%s", responderModule)
	}
	if strings.Contains(responderModule, "func EncodeHeartbeatRequest") || strings.Contains(responderModule, "func DecodeHeartbeatResponse") {
		t.Fatalf("responder module contains requester code:\n%s", responderModule)
	}
	if _, ok := filesByPath(responderFiles)["protocol.go"]; !ok {
		t.Fatalf("responder output should include protocol.go: %+v", responderFiles)
	}
}

func TestRenderRejectsInvalidSide(t *testing.T) {
	_, err := Render(testModel(), Options{Package: "protocolpb", Sides: []string{"mobile"}})
	if err == nil {
		t.Fatal("Render() error = nil, want invalid side error")
	}
}

func TestRenderRejectsInvalidRuntimeMode(t *testing.T) {
	_, err := Render(testModel(), Options{Package: "protocolpb", Sides: []string{"requester"}, Runtime: RuntimeMode("copy")})
	if err == nil {
		t.Fatal("Render() error = nil, want invalid runtime mode error")
	}
}

func TestRenderImportRuntimeRequiresExternalRuntimeImports(t *testing.T) {
	_, err := Render(testModel(), Options{Package: "protocolpb", Sides: []string{"requester"}, Runtime: RuntimeModeImport})
	if err == nil {
		t.Fatal("Render() error = nil, want missing runtime imports error")
	}
}

func TestRenderUsesGoImportOverrides(t *testing.T) {
	files, err := Render(testModel(), Options{
		Package: "protocolpb",
		Sides:   []string{"requester", "responder"},
		Runtime: RuntimeModeImport,
		Imports: ImportPaths{
			Frame:     "example.com/runtime/frame",
			Registry:  "example.com/runtime/registry",
			Proto:     "example.com/protobuf/proto",
			Context:   "example.com/context",
			FX:        "example.com/fx",
			Bootstrap: "example.com/bootstrap",
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	protocol := string(filesByPath(files)["protocol.go"].Content)
	module := string(filesByPath(files)["player.go"].Content)
	messages := string(filesByPath(files)["protocol_messages.go"].Content)
	all := protocol + module + messages
	for _, want := range []string{
		`"example.com/runtime/frame"`,
		`"example.com/runtime/registry"`,
		`"example.com/protobuf/proto"`,
	} {
		if !strings.Contains(all, want) {
			t.Fatalf("generated files missing import %q:\n%s\n%s\n%s", want, protocol, module, messages)
		}
	}
	if !strings.Contains(protocol, `bootstrap "example.com/bootstrap"`) || !strings.Contains(protocol, `"example.com/fx"`) || !strings.Contains(protocol, `"example.com/context"`) {
		t.Fatalf("protocol missing overridden framework imports:\n%s", protocol)
	}
	if strings.Contains(protocol+module+messages, "github.com/wxdqing/go-transformgen/runtime") {
		t.Fatalf("generated files still contain default transformgen runtime import")
	}
}

func TestRenderCanEmitRuntimeSupport(t *testing.T) {
	files, err := Render(testModel(), Options{Package: "protocolpb", Sides: []string{"requester", "responder"}, Runtime: RuntimeModeEmit})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(files)
	for _, path := range []string{"runtime_frame.go", "runtime_registry.go"} {
		if string(byPath[path].Content) == "" {
			t.Fatalf("missing emitted runtime file %s: %+v", path, files)
		}
	}
	all := string(byPath["protocol.go"].Content) + string(byPath["protocol_messages.go"].Content) + string(byPath["player.go"].Content)
	if strings.Contains(all, "github.com/wxdqing/go-transformgen/runtime") {
		t.Fatalf("runtime emit output should not import transformgen runtime:\n%s", all)
	}
	if !strings.Contains(string(byPath["runtime_frame.go"].Content), "type FrameCodec interface") {
		t.Fatalf("runtime_frame.go missing FrameCodec:\n%s", byPath["runtime_frame.go"].Content)
	}
	if !strings.Contains(string(byPath["runtime_registry.go"].Content), "type Registry interface") {
		t.Fatalf("runtime_registry.go missing Registry:\n%s", byPath["runtime_registry.go"].Content)
	}
	if strings.Contains(string(byPath["runtime_registry.go"].Content), "type DefaultRegistry struct") {
		t.Fatalf("runtime_registry.go should not export the concrete registry type:\n%s", byPath["runtime_registry.go"].Content)
	}
	if !strings.Contains(string(byPath["runtime_registry.go"].Content), "func NewRegistry() Registry") {
		t.Fatalf("runtime_registry.go should expose NewRegistry through the Registry interface:\n%s", byPath["runtime_registry.go"].Content)
	}
}

func testModel() *model.Model {
	return &model.Model{Modules: []model.Module{{
		Name:          "player",
		ConstName:     "ModelNamePlayer",
		InterfaceName: "Player",
		RPCs: []model.RPC{{
			Method:        "Heartbeat",
			Ctx:           "context.Context",
			CtxImportPath: "context",
			Request: descriptor.Message{
				ID:            1001,
				Kind:          descriptor.MessageKindRequest,
				FullName:      "transform.HeartbeatRequest",
				GoImportPath:  "github.com/wxdqing/go-transformgen/example/transform",
				GoPackageName: "transformpb",
				GoTypeName:    "HeartbeatRequest",
			},
			Response: descriptor.Message{
				ID:            1002,
				Kind:          descriptor.MessageKindResponse,
				FullName:      "transform.HeartbeatResponse",
				GoImportPath:  "github.com/wxdqing/go-transformgen/example/transform",
				GoPackageName: "transformpb",
				GoTypeName:    "HeartbeatResponse",
			},
		}},
		Notifies: []model.Notify{{
			Method:        "BattleFinished",
			Ctx:           "context.Context",
			CtxImportPath: "context",
			Message: descriptor.Message{
				ID:            2001,
				Kind:          descriptor.MessageKindNotify,
				FullName:      "transform.BattleFinishedNotify",
				GoImportPath:  "github.com/wxdqing/go-transformgen/example/transform",
				GoPackageName: "transformpb",
				GoTypeName:    "BattleFinishedNotify",
			},
		}},
	}}}
}
