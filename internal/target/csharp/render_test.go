package csharptarget

import (
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
)

func TestRenderRequesterWithRuntimeEmit(t *testing.T) {
	files, err := Render(testModel(), Options{
		Namespace: "Plan.Protocol",
		Sides:     []string{"requester"},
		Runtime:   RuntimeModeEmit,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(files)
	for _, path := range []string{"Frame.cs", "ProtocolRuntime.cs", "ProtocolMessages.cs", "PlayerRequester.cs"} {
		if byPath[path] == "" {
			t.Fatalf("missing %s: %+v", path, files)
		}
	}
	requester := byPath["PlayerRequester.cs"]
	for _, want := range []string{
		"namespace Plan.Protocol;",
		"public static class PlayerRequester",
		"public static byte[] EncodeHeartbeatRequest",
		"Plan.Transform.HeartbeatRequest request",
		"public static Plan.Transform.HeartbeatResponse DecodeHeartbeatResponse",
		"Plan.Transform.HeartbeatResponse.Parser.ParseFrom(payload)",
		"public static byte[] EncodeBattleFinishedNotify",
	} {
		if !strings.Contains(requester, want) {
			t.Fatalf("requester missing %q:\n%s", want, requester)
		}
	}
	if strings.Contains(requester, "interface IPlayerHandler") {
		t.Fatalf("requester should not contain responder code:\n%s", requester)
	}
	messages := byPath["ProtocolMessages.cs"]
	if !strings.Contains(messages, "public const uint HeartbeatRequest = 1001;") {
		t.Fatalf("messages missing constant:\n%s", messages)
	}
	frame := byPath["Frame.cs"]
	if !strings.Contains(frame, "public interface IFrameCodec") || !strings.Contains(frame, "public sealed class PacketFrameCodec") {
		t.Fatalf("frame runtime missing codec types:\n%s", frame)
	}
}

func TestRenderCSharpRejectsImportRuntime(t *testing.T) {
	_, err := Render(testModel(), Options{Namespace: "Plan.Protocol", Sides: []string{"requester"}, Runtime: RuntimeModeImport})
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

func TestRenderResponderDispatch(t *testing.T) {
	files, err := Render(testModel(), Options{
		Namespace: "Plan.Protocol",
		Sides:     []string{"responder"},
		Runtime:   RuntimeModeEmit,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	byPath := filesByPath(files)
	responder := byPath["PlayerResponder.cs"]
	for _, want := range []string{
		"public interface IPlayerHandler",
		"Plan.Transform.HeartbeatResponse Heartbeat(object ctx, Plan.Transform.HeartbeatRequest request);",
		"void BattleFinished(object ctx, Plan.Transform.BattleFinishedNotify message);",
		"public static IMessage DispatchRequest",
		"responseMessageId = MessageIds.HeartbeatResponse;",
		"return handler.Heartbeat(ctx, Plan.Transform.HeartbeatRequest.Parser.ParseFrom(payload));",
		"public static void DispatchNotify",
		"handler.BattleFinished(ctx, Plan.Transform.BattleFinishedNotify.Parser.ParseFrom(payload));",
	} {
		if !strings.Contains(responder, want) {
			t.Fatalf("responder missing %q:\n%s", want, responder)
		}
	}
	if strings.Contains(responder, "EncodeHeartbeatRequest") {
		t.Fatalf("responder should not contain requester code:\n%s", responder)
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
			Method: "Heartbeat",
			Request: descriptor.Message{
				ID:        1001,
				Kind:      descriptor.MessageKindRequest,
				FullName:  "transform.HeartbeatRequest",
				ProtoName: "HeartbeatRequest",
				CSharp:    descriptor.CSharpMessage{Namespace: "Plan.Transform", TypeName: "HeartbeatRequest"},
			},
			Response: descriptor.Message{
				ID:        1002,
				Kind:      descriptor.MessageKindResponse,
				FullName:  "transform.HeartbeatResponse",
				ProtoName: "HeartbeatResponse",
				CSharp:    descriptor.CSharpMessage{Namespace: "Plan.Transform", TypeName: "HeartbeatResponse"},
			},
		}},
		Notifies: []model.Notify{{
			Method: "BattleFinished",
			Message: descriptor.Message{
				ID:        2001,
				Kind:      descriptor.MessageKindNotify,
				FullName:  "transform.BattleFinishedNotify",
				ProtoName: "BattleFinishedNotify",
				CSharp:    descriptor.CSharpMessage{Namespace: "Plan.Transform", TypeName: "BattleFinishedNotify"},
			},
		}},
	}}}
}
