package protocolpb

import (
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	proto "google.golang.org/protobuf/proto"
)

const MessageIDBattleStateNotify uint32 = 114012388
const MessageIDSendChatResponse uint32 = 124399461
const MessageIDBattleFinishedNotify uint32 = 143096507
const MessageIDHeartbeatResponse uint32 = 168595187
const MessageIDChatMessageNotify uint32 = 170889542
const MessageIDStartBattleResponse uint32 = 171396577
const MessageIDStartBattleRequest uint32 = 234959079
const MessageIDSendChatRequest uint32 = 235223567
const MessageIDHeartbeatRequest uint32 = 259926425

func RegisterMessages(reg MessageRegistry) error {
	if err := reg.RegisterNotify(MessageMeta{ID: MessageIDBattleStateNotify, Kind: MessageKindNotify, FullName: "transform.example.BattleStateNotify"}, func() proto.Message { return &examplepb.BattleStateNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(MessageMeta{ID: MessageIDSendChatResponse, Kind: MessageKindResponse, FullName: "transform.example.SendChatResponse"}, func() proto.Message { return &examplepb.SendChatResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(MessageMeta{ID: MessageIDBattleFinishedNotify, Kind: MessageKindNotify, FullName: "transform.example.BattleFinishedNotify"}, func() proto.Message { return &examplepb.BattleFinishedNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(MessageMeta{ID: MessageIDHeartbeatResponse, Kind: MessageKindResponse, FullName: "transform.example.HeartbeatResponse"}, func() proto.Message { return &examplepb.HeartbeatResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterNotify(MessageMeta{ID: MessageIDChatMessageNotify, Kind: MessageKindNotify, FullName: "transform.example.ChatMessageNotify"}, func() proto.Message { return &examplepb.ChatMessageNotify{} }); err != nil {
		return err
	}
	if err := reg.RegisterResponse(MessageMeta{ID: MessageIDStartBattleResponse, Kind: MessageKindResponse, FullName: "transform.example.StartBattleResponse"}, func() proto.Message { return &examplepb.StartBattleResponse{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(MessageMeta{ID: MessageIDStartBattleRequest, Kind: MessageKindRequest, FullName: "transform.example.StartBattleRequest"}, func() proto.Message { return &examplepb.StartBattleRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(MessageMeta{ID: MessageIDSendChatRequest, Kind: MessageKindRequest, FullName: "transform.example.SendChatRequest"}, func() proto.Message { return &examplepb.SendChatRequest{} }); err != nil {
		return err
	}
	if err := reg.RegisterRequest(MessageMeta{ID: MessageIDHeartbeatRequest, Kind: MessageKindRequest, FullName: "transform.example.HeartbeatRequest"}, func() proto.Message { return &examplepb.HeartbeatRequest{} }); err != nil {
		return err
	}
	return nil
}
