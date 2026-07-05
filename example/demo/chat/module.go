package chat

import (
	"context"

	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	protocolpb "github.com/wxdqing/go-transformgen/example/transform/protocol"
)

type Module struct {
	MessageID uint64
}

func NewModule() protocolpb.HandlerModuleWithBean[*Module] {
	m := &Module{}
	return protocolpb.NewHandlerModuleWithBean(m)
}

func (m *Module) ModuleName() string { return protocolpb.ModelNameChat }
func (m *Module) Module() any        { return m }

func (m *Module) SendChat(_ context.Context, _ *examplepb.SendChatRequest) (*examplepb.SendChatResponse, error) {
	return &examplepb.SendChatResponse{MessageId: 8001}, nil
}

func (m *Module) ChatMessage(_ context.Context, msg *examplepb.ChatMessageNotify) error {
	m.MessageID = msg.GetMessageId()
	return nil
}
