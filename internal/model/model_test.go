package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/internal/define"
	"github.com/wxdqing/go-transformgen/internal/descriptor"
)

func TestBuildLinksRPCAndNotifyDefinitions(t *testing.T) {
	desc := NewDescriptorSetForTest(
		msg("transform.HeartbeatRequest", 1001, descriptor.MessageKindRequest),
		msg("transform.HeartbeatResponse", 1002, descriptor.MessageKindResponse),
		msg("transform.BattleFinishedNotify", 2001, descriptor.MessageKindNotify),
	)
	modules := []define.Module{{
		Name:    "player",
		Version: define.Version,
		RPCs: []define.RPC{{
			Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context", CtxImportPath: "context",
		}},
		Notifies: []define.Notify{{
			Method: "BattleFinished", Message: "transform.BattleFinishedNotify", Ctx: "context.Context", CtxImportPath: "context",
		}},
	}}

	model, err := Build(desc, modules)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(model.Modules) != 1 {
		t.Fatalf("len(modules) = %d, want 1", len(model.Modules))
	}
	module := model.Modules[0]
	if module.ConstName != "ModelNamePlayer" || module.InterfaceName != "Player" {
		t.Fatalf("module = %+v", module)
	}
	if module.RPCs[0].Request.ID != 1001 || module.RPCs[0].Response.ID != 1002 {
		t.Fatalf("rpc = %+v", module.RPCs[0])
	}
	if module.RPCs[0].CtxImportPath != "context" {
		t.Fatalf("rpc ctx import = %q, want context", module.RPCs[0].CtxImportPath)
	}
	if module.Notifies[0].Message.ID != 2001 {
		t.Fatalf("notify = %+v", module.Notifies[0])
	}
	if module.Notifies[0].CtxImportPath != "context" {
		t.Fatalf("notify ctx import = %q, want context", module.Notifies[0].CtxImportPath)
	}
}

func TestBuildRejectsMissingMessagesKindMismatchAndDuplicateRequest(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		desc := NewDescriptorSetForTest()
		_, err := Build(desc, []define.Module{{Name: "player", Version: define.Version, RPCs: []define.RPC{{
			Method: "Heartbeat", Request: "transform.Missing", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
		}}}})
		if !errors.Is(err, ErrMessageNotFound) {
			t.Fatalf("Build() error = %v, want ErrMessageNotFound", err)
		}
	})

	t.Run("kind", func(t *testing.T) {
		desc := NewDescriptorSetForTest(
			msg("transform.HeartbeatRequest", 1001, descriptor.MessageKindNotify),
			msg("transform.HeartbeatResponse", 1002, descriptor.MessageKindResponse),
		)
		_, err := Build(desc, []define.Module{{Name: "player", Version: define.Version, RPCs: []define.RPC{{
			Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
		}}}})
		if !errors.Is(err, ErrMessageKindMismatch) {
			t.Fatalf("Build() error = %v, want ErrMessageKindMismatch", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		desc := NewDescriptorSetForTest(
			msg("transform.HeartbeatRequest", 1001, descriptor.MessageKindRequest),
			msg("transform.HeartbeatResponse", 1002, descriptor.MessageKindResponse),
		)
		_, err := Build(desc, []define.Module{
			{Name: "player", Version: define.Version, RPCs: []define.RPC{{
				Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
			}}},
			{Name: "battle", Version: define.Version, RPCs: []define.RPC{{
				Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
			}}},
		})
		if !errors.Is(err, ErrDuplicateRequest) {
			t.Fatalf("Build() error = %v, want ErrDuplicateRequest", err)
		}
	})
}

func NewDescriptorSetForTest(messages ...descriptor.Message) *descriptor.Set {
	return descriptor.NewSet(messages...)
}

func msg(fullName string, id uint32, kind descriptor.MessageKind) descriptor.Message {
	return descriptor.Message{
		ID:            id,
		Kind:          kind,
		FullName:      fullName,
		GoImportPath:  "resource/protocol/src/transform",
		GoPackageName: "transformpb",
		GoTypeName:    fullName[strings.LastIndex(fullName, ".")+1:],
	}
}
