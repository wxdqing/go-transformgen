package protocolpb

import (
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	"github.com/wxdqing/go-transformgen/runtime/registry"
	"google.golang.org/protobuf/proto"
)

const MessageIDHeartbeatRequest uint32 = 1001
const MessageIDHeartbeatResponse uint32 = 1002
const MessageIDStartBattleRequest uint32 = 1101
const MessageIDStartBattleResponse uint32 = 1102
const MessageIDSendChatRequest uint32 = 1201
const MessageIDSendChatResponse uint32 = 1202
const MessageIDBattleFinishedNotify uint32 = 2001
const MessageIDBattleStateNotify uint32 = 2101
const MessageIDChatMessageNotify uint32 = 2201

func RegisterMessages(reg registry.MessageRegistry) error {
	if err := reg.RegisterRequest(registry.MessageMeta{ID: MessageIDHeartbeatRequest, Kind: registry.MessageKindRequest, FullName: "transform.example.HeartbeatRequest"}, func() proto.Message { return &examplepb.HeartbeatRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(registry.MessageMeta{ID: MessageIDHeartbeatResponse, Kind: registry.MessageKindResponse, FullName: "transform.example.HeartbeatResponse"}, func() proto.Message { return &examplepb.HeartbeatResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(registry.MessageMeta{ID: MessageIDStartBattleRequest, Kind: registry.MessageKindRequest, FullName: "transform.example.StartBattleRequest"}, func() proto.Message { return &examplepb.StartBattleRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(registry.MessageMeta{ID: MessageIDStartBattleResponse, Kind: registry.MessageKindResponse, FullName: "transform.example.StartBattleResponse"}, func() proto.Message { return &examplepb.StartBattleResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(registry.MessageMeta{ID: MessageIDSendChatRequest, Kind: registry.MessageKindRequest, FullName: "transform.example.SendChatRequest"}, func() proto.Message { return &examplepb.SendChatRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(registry.MessageMeta{ID: MessageIDSendChatResponse, Kind: registry.MessageKindResponse, FullName: "transform.example.SendChatResponse"}, func() proto.Message { return &examplepb.SendChatResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(registry.MessageMeta{ID: MessageIDBattleFinishedNotify, Kind: registry.MessageKindNotify, FullName: "transform.example.BattleFinishedNotify"}, func() proto.Message { return &examplepb.BattleFinishedNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(registry.MessageMeta{ID: MessageIDBattleStateNotify, Kind: registry.MessageKindNotify, FullName: "transform.example.BattleStateNotify"}, func() proto.Message { return &examplepb.BattleStateNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(registry.MessageMeta{ID: MessageIDChatMessageNotify, Kind: registry.MessageKindNotify, FullName: "transform.example.ChatMessageNotify"}, func() proto.Message { return &examplepb.ChatMessageNotify{} }); err != nil {
		return err
	}
	return nil
}
