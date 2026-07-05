package protocolpb

import (
	"context"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	"github.com/wxdqing/go-transformgen/runtime/frame"
	"github.com/wxdqing/go-transformgen/runtime/registry"
	"google.golang.org/protobuf/proto"
)

const ModelNameChat = "chat"

type Chat interface {
	SendChat(ctx context.Context, req *examplepb.SendChatRequest) (*examplepb.SendChatResponse, error)
	ChatMessage(ctx context.Context, msg *examplepb.ChatMessageNotify) error
}

func registerChatHandlers(reg registry.HandlerRegistry, impl Chat) error {
	if err := reg.RegisterRequestHandler(ModelNameChat, MessageIDSendChatRequest, MessageIDSendChatResponse, func(ctx any, req proto.Message) (proto.Message, error) {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return nil, registry.ErrInvalidContextType
		}
		typedReq, ok := req.(*examplepb.SendChatRequest)
		if !ok {
			return nil, registry.ErrInvalidMessageType
		}
		return impl.SendChat(typedCtx, typedReq)
	}); err != nil {
		return err
	}
	if err := reg.RegisterNotifyHandler(ModelNameChat, MessageIDChatMessageNotify, func(ctx any, msg proto.Message) error {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return registry.ErrInvalidContextType
		}
		typedMsg, ok := msg.(*examplepb.ChatMessageNotify)
		if !ok {
			return registry.ErrInvalidMessageType
		}
		return impl.ChatMessage(typedCtx, typedMsg)
	}); err != nil {
		return err
	}
	return nil
}

func EncodeSendChatRequest(codec frame.FrameCodec, requestID uint64, req *examplepb.SendChatRequest) ([]byte, func(), error) {
	return PackMessage(codec, frame.Head{MessageID: MessageIDSendChatRequest, RequestID: requestID}, req)
}

func DecodeSendChatResponse(messageID uint32, payload []byte) (*examplepb.SendChatResponse, error) {
	if messageID != MessageIDSendChatResponse {
		return nil, registry.ErrMessageKindMismatch
	}
	var resp examplepb.SendChatResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
func EncodeChatMessageNotify(codec frame.FrameCodec, msg *examplepb.ChatMessageNotify) ([]byte, func(), error) {
	return PackMessage(codec, frame.Head{MessageID: MessageIDChatMessageNotify}, msg)
}
