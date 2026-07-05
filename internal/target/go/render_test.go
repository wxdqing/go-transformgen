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

	files, err := Render(m, Options{Package: "protocolpb", Sides: []string{"responder"}})
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
		"PackMessage(codec, frame.Head{MessageID: MessageIDHeartbeatRequest, RequestID: requestID}, req)",
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
