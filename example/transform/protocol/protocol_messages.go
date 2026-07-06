package protocolpb

import (
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	proto "google.golang.org/protobuf/proto"
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

func RegisterMessages(reg MessageRegistry) error {
	if err := reg.RegisterRequest(MessageMeta{ID: MessageIDHeartbeatRequest, Kind: MessageKindRequest, FullName: "transform.example.HeartbeatRequest"}, func() proto.Message { return &examplepb.HeartbeatRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(MessageMeta{ID: MessageIDHeartbeatResponse, Kind: MessageKindResponse, FullName: "transform.example.HeartbeatResponse"}, func() proto.Message { return &examplepb.HeartbeatResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(MessageMeta{ID: MessageIDStartBattleRequest, Kind: MessageKindRequest, FullName: "transform.example.StartBattleRequest"}, func() proto.Message { return &examplepb.StartBattleRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(MessageMeta{ID: MessageIDStartBattleResponse, Kind: MessageKindResponse, FullName: "transform.example.StartBattleResponse"}, func() proto.Message { return &examplepb.StartBattleResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(MessageMeta{ID: MessageIDSendChatRequest, Kind: MessageKindRequest, FullName: "transform.example.SendChatRequest"}, func() proto.Message { return &examplepb.SendChatRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(MessageMeta{ID: MessageIDSendChatResponse, Kind: MessageKindResponse, FullName: "transform.example.SendChatResponse"}, func() proto.Message { return &examplepb.SendChatResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(MessageMeta{ID: MessageIDBattleFinishedNotify, Kind: MessageKindNotify, FullName: "transform.example.BattleFinishedNotify"}, func() proto.Message { return &examplepb.BattleFinishedNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(MessageMeta{ID: MessageIDBattleStateNotify, Kind: MessageKindNotify, FullName: "transform.example.BattleStateNotify"}, func() proto.Message { return &examplepb.BattleStateNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(MessageMeta{ID: MessageIDChatMessageNotify, Kind: MessageKindNotify, FullName: "transform.example.ChatMessageNotify"}, func() proto.Message { return &examplepb.ChatMessageNotify{} }); err != nil {
		return err
	}
	return nil
}
