package battle

import (
	"context"

	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	protocolpb "github.com/wxdqing/go-transformgen/example/transform/protocol"
)

type Module struct {
	State string
}

func NewModule() protocolpb.HandlerModuleWithBean[*Module] {
	m := &Module{}
	return protocolpb.NewHandlerModuleWithBean(m)
}

func (m *Module) ModuleName() string { return protocolpb.ModelNameBattle }
func (m *Module) Module() any        { return m }

func (m *Module) StartBattle(_ context.Context, _ *examplepb.StartBattleRequest) (*examplepb.StartBattleResponse, error) {
	return &examplepb.StartBattleResponse{BattleId: 9001, Accepted: true}, nil
}

func (m *Module) BattleState(_ context.Context, msg *examplepb.BattleStateNotify) error {
	m.State = msg.GetState()
	return nil
}
