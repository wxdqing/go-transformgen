package protocolpb

import (
	"context"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	"github.com/wxdqing/go-transformgen/runtime/frame"
	"github.com/wxdqing/go-transformgen/runtime/registry"
	"google.golang.org/protobuf/proto"
)

const ModelNamePlayer = "player"

type Player interface {
	Heartbeat(ctx context.Context, req *examplepb.HeartbeatRequest) (*examplepb.HeartbeatResponse, error)
	BattleFinished(ctx context.Context, msg *examplepb.BattleFinishedNotify) error
}

func registerPlayerHandlers(reg registry.HandlerRegistry, impl Player) error {
	if err := reg.RegisterRequestHandler(ModelNamePlayer, MessageIDHeartbeatRequest, MessageIDHeartbeatResponse, func(ctx any, req proto.Message) (proto.Message, error) {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return nil, registry.ErrInvalidContextType
		}
		typedReq, ok := req.(*examplepb.HeartbeatRequest)
		if !ok {
			return nil, registry.ErrInvalidMessageType
		}
		return impl.Heartbeat(typedCtx, typedReq)
	}); err != nil {
		return err
	}
	if err := reg.RegisterNotifyHandler(ModelNamePlayer, MessageIDBattleFinishedNotify, func(ctx any, msg proto.Message) error {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return registry.ErrInvalidContextType
		}
		typedMsg, ok := msg.(*examplepb.BattleFinishedNotify)
		if !ok {
			return registry.ErrInvalidMessageType
		}
		return impl.BattleFinished(typedCtx, typedMsg)
	}); err != nil {
		return err
	}
	return nil
}

func EncodeHeartbeatRequest(codec frame.FrameCodec, requestID uint64, req *examplepb.HeartbeatRequest) ([]byte, func(), error) {
	return PackMessage(codec, frame.Head{MessageID: MessageIDHeartbeatRequest, RequestID: requestID}, req)
}

func DecodeHeartbeatResponse(messageID uint32, payload []byte) (*examplepb.HeartbeatResponse, error) {
	if messageID != MessageIDHeartbeatResponse {
		return nil, registry.ErrMessageKindMismatch
	}
	var resp examplepb.HeartbeatResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
func EncodeBattleFinishedNotify(codec frame.FrameCodec, msg *examplepb.BattleFinishedNotify) ([]byte, func(), error) {
	return PackMessage(codec, frame.Head{MessageID: MessageIDBattleFinishedNotify}, msg)
}
