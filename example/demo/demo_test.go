package demo

import (
	"context"
	"testing"

	"github.com/wxdqing/go-transformgen/example/demo/battle"
	"github.com/wxdqing/go-transformgen/example/demo/chat"
	"github.com/wxdqing/go-transformgen/example/demo/player"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	protocolpb "github.com/wxdqing/go-transformgen/example/transform/protocol"
	"github.com/wxdqing/go-transformgen/runtime/registry"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"google.golang.org/protobuf/proto"
)

func TestDemoFxGroupRegistersProtocolModules(t *testing.T) {
	var protocol *protocolpb.Module
	var playerMod *player.Module
	var battleMod *battle.Module
	var chatMod *chat.Module

	app := fxtest.New(
		t,
		fx.Provide(
			protocolpb.Provider{}.Register(),
			player.Provider{}.Register(),
			battle.Provider{}.Register(),
			chat.Provider{}.Register(),
		),
		fx.Populate(&protocol, &playerMod, &battleMod, &chatMod),
	)
	app.RequireStart()
	defer app.RequireStop()

	if protocol.Registry != nil {
		t.Fatal("protocol registry should be initialized by Start")
	}
	start, ok := protocolpb.Provider{}.OnStart().(func(context.Context, *protocolpb.Module) error)
	if !ok {
		t.Fatalf("Provider.OnStart() = %T", protocolpb.Provider{}.OnStart())
	}
	if err := start(context.Background(), protocol); err != nil {
		t.Fatalf("protocol start: %v", err)
	}

	assertRequest(t, protocol.Registry, protocolpb.MessageIDHeartbeatRequest, &examplepb.HeartbeatRequest{ClientTime: 10, Sequence: 2}, protocolpb.MessageIDHeartbeatResponse)
	assertRequest(t, protocol.Registry, protocolpb.MessageIDStartBattleRequest, &examplepb.StartBattleRequest{PlayerId: 1, StageId: "stage"}, protocolpb.MessageIDStartBattleResponse)
	assertRequest(t, protocol.Registry, protocolpb.MessageIDSendChatRequest, &examplepb.SendChatRequest{ChannelId: 8, Content: "hello"}, protocolpb.MessageIDSendChatResponse)

	dispatchNotify(t, protocol.Registry, protocolpb.MessageIDBattleFinishedNotify, &examplepb.BattleFinishedNotify{BattleId: 42})
	if playerMod.BattleID != 42 {
		t.Fatalf("player battle id = %d, want 42", playerMod.BattleID)
	}
	dispatchNotify(t, protocol.Registry, protocolpb.MessageIDBattleStateNotify, &examplepb.BattleStateNotify{BattleId: 42, State: "done"})
	if battleMod.State != "done" {
		t.Fatalf("battle state = %q, want done", battleMod.State)
	}
	dispatchNotify(t, protocol.Registry, protocolpb.MessageIDChatMessageNotify, &examplepb.ChatMessageNotify{ChannelId: 8, MessageId: 77, Content: "hi"})
	if chatMod.MessageID != 77 {
		t.Fatalf("chat message id = %d, want 77", chatMod.MessageID)
	}
}

func assertRequest(t *testing.T, reg registry.HandlerRegistry, requestID uint32, req proto.Message, responseID uint32) {
	t.Helper()
	payload, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	_, gotResponseID, err := reg.DispatchRequest(context.Background(), requestID, payload)
	if err != nil {
		t.Fatalf("DispatchRequest(%d) error = %v", requestID, err)
	}
	if gotResponseID != responseID {
		t.Fatalf("response id = %d, want %d", gotResponseID, responseID)
	}
}

func dispatchNotify(t *testing.T, reg registry.HandlerRegistry, messageID uint32, msg proto.Message) {
	t.Helper()
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.DispatchNotify(context.Background(), messageID, payload); err != nil {
		t.Fatalf("DispatchNotify(%d) error = %v", messageID, err)
	}
}
