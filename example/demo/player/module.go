package player

import (
	"context"

	examplepb "github.com/wxdqing/go-transformgen/example/transform"
	protocolpb "github.com/wxdqing/go-transformgen/example/transform/protocol"
)

type Module struct {
	BattleID uint64
}

func NewModule() protocolpb.HandlerModuleWithBean[*Module] {
	m := &Module{}
	return protocolpb.NewHandlerModuleWithBean(m)
}

func (m *Module) ModuleName() string { return protocolpb.ModelNamePlayer }
func (m *Module) Module() any        { return m }

func (m *Module) Heartbeat(_ context.Context, req *examplepb.HeartbeatRequest) (*examplepb.HeartbeatResponse, error) {
	return &examplepb.HeartbeatResponse{
		ServerTime: req.GetClientTime() + 1,
		ClientTime: req.GetClientTime(),
		Sequence:   req.GetSequence(),
	}, nil
}

func (m *Module) BattleFinished(_ context.Context, msg *examplepb.BattleFinishedNotify) error {
	m.BattleID = msg.GetBattleId()
	return nil
}
