package cpptarget

import (
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

// testModel builds one player module with an RPC and a notify.
func testModel() *model.Model {
	return &model.Model{Modules: []model.Module{{
		Name:          "player",
		ConstName:     "ModelNamePlayer",
		InterfaceName: "Player",
		RPCs: []model.RPC{{
			Method:   "Heartbeat",
			Request:  msg("transform.HeartbeatRequest", 200012345, descriptor.MessageKindRequest, "heartbeat.proto"),
			Response: msg("transform.HeartbeatResponse", 100012345, descriptor.MessageKindResponse, "heartbeat.proto"),
		}},
		Notifies: []model.Notify{{
			Method:  "BattleFinished",
			Message: msg("transform.BattleFinishedNotify", 100054321, descriptor.MessageKindNotify, "battle.proto"),
		}},
	}}}
}

func msg(fullName string, id uint32, kind descriptor.MessageKind, sourceFile string) descriptor.Message {
	short := fullName[strings.LastIndex(fullName, ".")+1:]
	return descriptor.Message{
		ID:           id,
		Kind:         kind,
		FullName:     fullName,
		ProtoPackage: "transform",
		ProtoName:    short,
		SourceFile:   "example/transform/" + sourceFile,
	}
}

func filesByPath(files []File) map[string]string {
	out := make(map[string]string, len(files))
	for _, file := range files {
		out[file.Path] = string(file.Content)
	}
	return out
}

func TestRenderEmitsEnumsFactoryAndTraits(t *testing.T) {
	files, err := Render(testModel(), Options{Namespace: "transform"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(files)
	header := byPath["protocol_messages.hpp"]
	for _, want := range []string{
		"namespace transform {",
		"enum class EMsgToServerType : std::uint32_t {",
		"HeartbeatRequest = 200012345,",
		"enum class EMsgToClientType : std::uint32_t {",
		"HeartbeatResponse = 100012345,",
		"BattleFinishedNotify = 100054321,",
		"constexpr std::uint32_t ToMessageId(EMsgToServerType type) noexcept",
		"std::unique_ptr<google::protobuf::Message> CreateMessage(std::uint32_t message_id);",
		"struct MessageId<::transform::HeartbeatRequest> {",
		"static constexpr std::uint32_t value = 200012345;",
		"inline constexpr std::uint32_t MessageIdOf = MessageId<Message>::value;",
		"#include \"heartbeat.pb.h\"",
		"#include \"battle.pb.h\"",
	} {
		if !strings.Contains(header, want) {
			t.Fatalf("protocol_messages.hpp missing %q:\n%s", want, header)
		}
	}
	source := byPath["protocol_messages.cpp"]
	for _, want := range []string{
		"{200012345u, MessageKind::Request, \"transform.HeartbeatRequest\"},",
		"case 200012345u:",
		"return std::make_unique<::transform::HeartbeatRequest>();",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("protocol_messages.cpp missing %q:\n%s", want, source)
		}
	}
}

func TestRenderModuleNamespaceWithoutModulePrefix(t *testing.T) {
	files, err := Render(testModel(), Options{Namespace: "transform"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(files)
	requester := byPath["player_requester.hpp"]
	responder := byPath["player_responder.hpp"]
	for _, want := range []string{
		"namespace transform::player {",
		"class Requester {",
		"bool SendHeartbeat(const PacketContext& context,",
	} {
		if !strings.Contains(requester, want) {
			t.Fatalf("player_requester.hpp missing %q:\n%s", want, requester)
		}
	}
	for _, want := range []string{
		"namespace transform::player {",
		"class ResponderHandler {",
		"class Responder {",
		"virtual bool Heartbeat(const PacketContext& context,",
		"bool SendBattleFinished(const PacketContext& context,",
	} {
		if !strings.Contains(responder, want) {
			t.Fatalf("player_responder.hpp missing %q:\n%s", want, responder)
		}
	}
	// Module classes must not repeat the module name as a prefix.
	for _, banned := range []string{"PlayerRequester", "PlayerResponder"} {
		if strings.Contains(requester, banned) || strings.Contains(responder, banned) {
			t.Fatalf("generated code contains banned name %q", banned)
		}
	}

	// Responses preserve request_id but clear the uplink sequence; notifies
	// clear both correlation fields before endpoint-side sequence allocation.
	responderSource := byPath["player_responder.cpp"]
	for _, want := range []string{
		"PacketContext reply_context = context;",
		"reply_context.packet_seq = 0;",
		"PacketContext notify_context = context;",
		"notify_context.request_id = 0;",
		"notify_context.packet_seq = 0;",
	} {
		if !strings.Contains(responderSource, want) {
			t.Fatalf("player_responder.cpp missing %q:\n%s", want, responderSource)
		}
	}
}

func TestRenderRespectsSides(t *testing.T) {
	requesterOnly, err := Render(testModel(), Options{Namespace: "transform", Sides: []string{"requester"}})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(requesterOnly)
	if byPath["player_requester.hpp"] == "" {
		t.Fatal("requester side missing player_requester.hpp")
	}
	if byPath["player_responder.hpp"] != "" {
		t.Fatal("requester side should not emit player_responder.hpp")
	}

	responderOnly, err := Render(testModel(), Options{Namespace: "transform", Sides: []string{"responder"}})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath = filesByPath(responderOnly)
	if byPath["player_responder.hpp"] == "" {
		t.Fatal("responder side missing player_responder.hpp")
	}
	if byPath["player_requester.hpp"] != "" {
		t.Fatal("responder side should not emit player_requester.hpp")
	}
}

func TestRenderProtoIncludePrefix(t *testing.T) {
	files, err := Render(testModel(), Options{Namespace: "transform", ProtoIncludePrefix: "protos/pb"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	header := filesByPath(files)["protocol_messages.hpp"]
	if !strings.Contains(header, "#include \"protos/pb/heartbeat.pb.h\"") {
		t.Fatalf("protocol_messages.hpp missing prefixed include:\n%s", header)
	}
}

func TestRenderRejectsImportRuntimeAndMissingNamespace(t *testing.T) {
	if _, err := Render(testModel(), Options{Namespace: "transform", Runtime: RuntimeModeImport}); err == nil {
		t.Fatal("Render() error = nil, want import runtime rejection")
	}
	if _, err := Render(testModel(), Options{}); err == nil {
		t.Fatal("Render() error = nil, want missing namespace rejection")
	}
}

func TestRenderRejectsNestedMessages(t *testing.T) {
	nested := testModel()
	nested.Modules[0].RPCs[0].Request = descriptor.Message{
		ID:           1,
		Kind:         descriptor.MessageKindRequest,
		FullName:     "transform.Outer.InnerRequest",
		ProtoPackage: "transform",
		ProtoName:    "InnerRequest",
		SourceFile:   "example/transform/outer.proto",
	}
	if _, err := Render(nested, Options{Namespace: "transform"}); err == nil {
		t.Fatal("Render() error = nil, want nested message rejection")
	}
}
