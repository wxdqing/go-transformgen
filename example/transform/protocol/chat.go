package protocolpb

import (
	"context"
	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	proto "google.golang.org/protobuf/proto"
)

const ModelNameChat = "chat"

type Chat interface {
	SendChat(ctx context.Context, req *examplepb.SendChatRequest) (*examplepb.SendChatResponse, error)
	ChatMessage(ctx context.Context, msg *examplepb.ChatMessageNotify) error
}

func registerChatHandlers(reg HandlerRegistry, impl Chat) error {
	if err := reg.RegisterRequestHandler(ModelNameChat, MessageIDSendChatRequest, MessageIDSendChatResponse, func(ctx any, req proto.Message) (proto.Message, error) {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return nil, ErrInvalidContextType
		}
		typedReq, ok := req.(*examplepb.SendChatRequest)
		if !ok {
			return nil, ErrInvalidMessageType
		}
		return impl.SendChat(typedCtx, typedReq)
	}); err != nil {
		return err
	}
	if err := reg.RegisterNotifyHandler(ModelNameChat, MessageIDChatMessageNotify, func(ctx any, msg proto.Message) error {
		typedCtx, ok := ctx.(context.Context)
		if !ok {
			return ErrInvalidContextType
		}
		typedMsg, ok := msg.(*examplepb.ChatMessageNotify)
		if !ok {
			return ErrInvalidMessageType
		}
		return impl.ChatMessage(typedCtx, typedMsg)
	}); err != nil {
		return err
	}
	return nil
}
func EncodeSendChatRequest(codec FrameCodec, requestID uint64, req *examplepb.SendChatRequest) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(req)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(Head{MessageID: MessageIDSendChatRequest, RequestID: requestID}, body)
}

func DecodeSendChatResponse(messageID uint32, payload []byte) (*examplepb.SendChatResponse, error) {
	if messageID != MessageIDSendChatResponse {
		return nil, ErrMessageKindMismatch
	}
	var resp examplepb.SendChatResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
func EncodeChatMessageNotify(codec FrameCodec, msg *examplepb.ChatMessageNotify) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(msg)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(Head{MessageID: MessageIDChatMessageNotify}, body)
}
