package protocolpb

import (
	context "context"
	"fmt"

	bootstrap "gitee.com/wxdqing/fx-bootstrap"
	fx "go.uber.org/fx"
	proto "google.golang.org/protobuf/proto"
)

type HandlerModule interface {
	ModuleName() string
	Module() any
}

type HandlerModuleOut struct {
	fx.Out

	Module HandlerModule `group:"transformgen_handler_modules"`
}

type HandlerModuleWithBean[T HandlerModule] struct {
	fx.Out

	Module HandlerModule `group:"transformgen_handler_modules"`
	Self   T
}

func NewHandlerModule(module HandlerModule) HandlerModuleOut {
	return HandlerModuleOut{Module: module}
}

func NewHandlerModuleWithBean[T HandlerModule](module T) HandlerModuleWithBean[T] {
	return HandlerModuleWithBean[T]{Module: module, Self: module}
}

type moduleParams struct {
	fx.In

	Modules []HandlerModule `group:"transformgen_handler_modules"`
}

type Provider struct {
	bootstrap.NopHook
	Codec FrameCodec
}

var _ bootstrap.Provider = Provider{}

func (p Provider) Register() any {
	return func(pms moduleParams) *Module {
		return NewModule(p.Codec, pms)
	}
}

func (Provider) OnStart() any {
	return func(ctx context.Context, m *Module) error {
		return m.Start(ctx)
	}
}

type Module struct {
	Registry Registry
	Codec    FrameCodec
	modules  []HandlerModule
}

type Protocol = Module

func NewModule(codec FrameCodec, p moduleParams) *Module {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	return &Module{Codec: codec, modules: p.Modules}
}

func NewProtocol(codec FrameCodec) (*Protocol, error) {
	m := NewModule(codec, moduleParams{})
	if err := m.Start(context.Background()); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) Start(_ context.Context) error {
	if m == nil {
		return fmt.Errorf("%w: nil protocol module", ErrHandlerNotFound)
	}
	reg := NewRegistry()
	if err := RegisterMessages(reg); err != nil {
		return err
	}
	m.Registry = reg
	for _, module := range m.modules {
		if module == nil {
			return fmt.Errorf("%w: nil handler module", ErrInvalidMessageType)
		}
		if err := m.RegisterHandlers(module.ModuleName(), module.Module()); err != nil {
			return err
		}
	}
	return nil
}

func (m *Module) RegisterHandlers(modelName string, impl any) error {
	if m == nil || m.Registry == nil {
		return fmt.Errorf("%w: nil protocol registry", ErrHandlerNotFound)
	}
	return RegisterHandlers(m.Registry, modelName, impl)
}

func (m *Module) PackMessage(head Head, msg proto.Message) ([]byte, func(), error) {
	if m == nil {
		return nil, func() {}, fmt.Errorf("%w: nil protocol", ErrInvalidMessageType)
	}
	return PackMessage(m.Codec, head, msg)
}

func PackMessage(codec FrameCodec, head Head, msg proto.Message) ([]byte, func(), error) {
	if codec == nil {
		codec = PacketFrameCodec{}
	}
	body, err := proto.Marshal(msg)
	if err != nil {
		return nil, func() {}, err
	}
	return codec.EncodeFrame(head, body)
}

func RegisterHandlers(reg HandlerRegistry, modelName string, impl any) error {
	switch modelName {
	case ModelNameBattle:
		if handler, ok := impl.(Battle); ok {
			return registerBattleHandlers(reg, handler)
		}
		return fmt.Errorf("%w: %s", ErrInvalidMessageType, modelName)
	case ModelNameChat:
		if handler, ok := impl.(Chat); ok {
			return registerChatHandlers(reg, handler)
		}
		return fmt.Errorf("%w: %s", ErrInvalidMessageType, modelName)
	case ModelNamePlayer:
		if handler, ok := impl.(Player); ok {
			return registerPlayerHandlers(reg, handler)
		}
		return fmt.Errorf("%w: %s", ErrInvalidMessageType, modelName)
	default:
		return fmt.Errorf("%w: model %s", ErrHandlerNotFound, modelName)
	}
}
