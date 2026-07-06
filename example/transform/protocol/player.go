package protocolpb

import (
	"context"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	proto "google.golang.org/protobuf/proto"
)

const ModelNamePlayer = "player"

type Player interface {
	Heartbeat(ctx context.Context, req *examplepb.HeartbeatRequest) (*examplepb.HeartbeatResponse, error)
	BattleFinished(ctx context.Context, msg *examplepb.BattleFinishedNotify) error
}

func registerPlayerHandlers(reg HandlerRegistry, impl Player) error {
	if err := reg.RegisterRequestHandler(ModelNamePlayer, MessageIDHeartbeatRequest, MessageIDHeartbeatResponse, func(ctx any, req proto.Message) (proto.Message, error) {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return nil, ErrInvalidContextType
		}
		typedReq, ok := req.(*examplepb.HeartbeatRequest)
		if !ok {
			return nil, ErrInvalidMessageType
		}
		return impl.Heartbeat(typedCtx, typedReq)
	}); err != nil {
		return err
	}
	if err := reg.RegisterNotifyHandler(ModelNamePlayer, MessageIDBattleFinishedNotify, func(ctx any, msg proto.Message) error {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return ErrInvalidContextType
		}
		typedMsg, ok := msg.(*examplepb.BattleFinishedNotify)
		if !ok {
			return ErrInvalidMessageType
		}
		return impl.BattleFinished(typedCtx, typedMsg)
	}); err != nil {
		return err
	}
	return nil
}
func EncodeHeartbeatRequest(codec FrameCodec, requestID uint64, req *examplepb.HeartbeatRequest) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(req)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(Head{MessageID: MessageIDHeartbeatRequest, RequestID: requestID}, body)
}

func DecodeHeartbeatResponse(messageID uint32, payload []byte) (*examplepb.HeartbeatResponse, error) {
	if messageID != MessageIDHeartbeatResponse {
		return nil, ErrMessageKindMismatch
	}
	var resp examplepb.HeartbeatResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
func EncodeBattleFinishedNotify(codec FrameCodec, msg *examplepb.BattleFinishedNotify) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(msg)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(Head{MessageID: MessageIDBattleFinishedNotify}, body)
}
