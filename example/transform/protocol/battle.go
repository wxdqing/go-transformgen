package protocolpb

import (
	"context"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	"github.com/wxdqing/go-transformgen/runtime/frame"
	"github.com/wxdqing/go-transformgen/runtime/registry"
	"google.golang.org/protobuf/proto"
)

const ModelNameBattle = "battle"

type Battle interface {
	StartBattle(ctx context.Context, req *examplepb.StartBattleRequest) (*examplepb.StartBattleResponse, error)
	BattleState(ctx context.Context, msg *examplepb.BattleStateNotify) error
}

func registerBattleHandlers(reg registry.HandlerRegistry, impl Battle) error {
	if err := reg.RegisterRequestHandler(ModelNameBattle, MessageIDStartBattleRequest, MessageIDStartBattleResponse, func(ctx any, req proto.Message) (proto.Message, error) {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return nil, registry.ErrInvalidContextType
		}
		typedReq, ok := req.(*examplepb.StartBattleRequest)
		if !ok {
			return nil, registry.ErrInvalidMessageType
		}
		return impl.StartBattle(typedCtx, typedReq)
	}); err != nil {
		return err
	}
	if err := reg.RegisterNotifyHandler(ModelNameBattle, MessageIDBattleStateNotify, func(ctx any, msg proto.Message) error {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return registry.ErrInvalidContextType
		}
		typedMsg, ok := msg.(*examplepb.BattleStateNotify)
		if !ok {
			return registry.ErrInvalidMessageType
		}
		return impl.BattleState(typedCtx, typedMsg)
	}); err != nil {
		return err
	}
	return nil
}

func EncodeStartBattleRequest(codec frame.FrameCodec, requestID uint64, req *examplepb.StartBattleRequest) ([]byte, func(), error) {
	return PackMessage(codec, frame.Head{MessageID: MessageIDStartBattleRequest, RequestID: requestID}, req)
}

func DecodeStartBattleResponse(messageID uint32, payload []byte) (*examplepb.StartBattleResponse, error) {
	if messageID != MessageIDStartBattleResponse {
		return nil, registry.ErrMessageKindMismatch
	}
	var resp examplepb.StartBattleResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
func EncodeBattleStateNotify(codec frame.FrameCodec, msg *examplepb.BattleStateNotify) ([]byte, func(), error) {
	return PackMessage(codec, frame.Head{MessageID: MessageIDBattleStateNotify}, msg)
}
