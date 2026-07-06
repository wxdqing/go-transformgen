package protocolpb

import (
	"context"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	proto "google.golang.org/protobuf/proto"
)

const ModelNameBattle = "battle"

type Battle interface {
	StartBattle(ctx context.Context, req *examplepb.StartBattleRequest) (*examplepb.StartBattleResponse, error)
	BattleState(ctx context.Context, msg *examplepb.BattleStateNotify) error
}

func registerBattleHandlers(reg HandlerRegistry, impl Battle) error {
	if err := reg.RegisterRequestHandler(ModelNameBattle, MessageIDStartBattleRequest, MessageIDStartBattleResponse, func(ctx any, req proto.Message) (proto.Message, error) {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return nil, ErrInvalidContextType
		}
		typedReq, ok := req.(*examplepb.StartBattleRequest)
		if !ok {
			return nil, ErrInvalidMessageType
		}
		return impl.StartBattle(typedCtx, typedReq)
	}); err != nil {
		return err
	}
	if err := reg.RegisterNotifyHandler(ModelNameBattle, MessageIDBattleStateNotify, func(ctx any, msg proto.Message) error {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return ErrInvalidContextType
		}
		typedMsg, ok := msg.(*examplepb.BattleStateNotify)
		if !ok {
			return ErrInvalidMessageType
		}
		return impl.BattleState(typedCtx, typedMsg)
	}); err != nil {
		return err
	}
	return nil
}
func EncodeStartBattleRequest(codec FrameCodec, requestID uint64, req *examplepb.StartBattleRequest) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(req)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(Head{MessageID: MessageIDStartBattleRequest, RequestID: requestID}, body)
}

func DecodeStartBattleResponse(messageID uint32, payload []byte) (*examplepb.StartBattleResponse, error) {
	if messageID != MessageIDStartBattleResponse {
		return nil, ErrMessageKindMismatch
	}
	var resp examplepb.StartBattleResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
func EncodeBattleStateNotify(codec FrameCodec, msg *examplepb.BattleStateNotify) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(msg)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(Head{MessageID: MessageIDBattleStateNotify}, body)
}
