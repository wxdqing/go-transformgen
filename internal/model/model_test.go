package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/wxdqing/go-transformgen/internal/define"
	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/msgid"
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

	wantReq := msgid.Compute("HeartbeatRequest", true)
	wantResp := msgid.Compute("HeartbeatResponse", false)
	wantNotify := msgid.Compute("BattleFinishedNotify", false)

	model, nextLock, err := Build(desc, modules, nil)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if nextLock["HeartbeatRequest"] != wantReq || nextLock["HeartbeatResponse"] != wantResp || nextLock["BattleFinishedNotify"] != wantNotify {
		t.Fatalf("nextLock = %#v", nextLock)
	}
	if len(model.Modules) != 1 {
		t.Fatalf("len(modules) = %d, want 1", len(model.Modules))
	}
	module := model.Modules[0]
	if module.ConstName != "ModelNamePlayer" || module.InterfaceName != "Player" {
		t.Fatalf("module = %+v", module)
	}
	if module.RPCs[0].Request.ID != wantReq || module.RPCs[0].Response.ID != wantResp {
		t.Fatalf("rpc ids = %d/%d, want %d/%d", module.RPCs[0].Request.ID, module.RPCs[0].Response.ID, wantReq, wantResp)
	}
	if module.RPCs[0].CtxImportPath != "context" {
		t.Fatalf("rpc ctx import = %q, want context", module.RPCs[0].CtxImportPath)
	}
	if module.Notifies[0].Message.ID != wantNotify {
		t.Fatalf("notify id = %d, want %d", module.Notifies[0].Message.ID, wantNotify)
	}
	if module.Notifies[0].CtxImportPath != "context" {
		t.Fatalf("notify ctx import = %q, want context", module.Notifies[0].CtxImportPath)
	}
}

func TestBuildPreservesLockedIDsWhenAddingMessages(t *testing.T) {
	// Simulate a prior collision: BattleFinishedNotify was bumped off its natural hash.
	naturalNotify := msgid.Compute("BattleFinishedNotify", false)
	lockedNotify := naturalNotify + 1
	locked := map[string]uint32{
		"HeartbeatRequest":     msgid.Compute("HeartbeatRequest", true),
		"HeartbeatResponse":    msgid.Compute("HeartbeatResponse", false),
		"BattleFinishedNotify": lockedNotify,
	}

	desc := NewDescriptorSetForTest(
		msg("transform.HeartbeatRequest", 0, descriptor.MessageKindRequest),
		msg("transform.HeartbeatResponse", 0, descriptor.MessageKindResponse),
		msg("transform.BattleFinishedNotify", 0, descriptor.MessageKindNotify),
		msg("transform.AaaNewNotify", 0, descriptor.MessageKindNotify),
	)
	modules := []define.Module{{
		Name:    "player",
		Version: define.Version,
		RPCs: []define.RPC{{
			Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
		}},
		Notifies: []define.Notify{
			{Method: "BattleFinished", Message: "transform.BattleFinishedNotify", Ctx: "context.Context"},
			{Method: "AaaNew", Message: "transform.AaaNewNotify", Ctx: "context.Context"},
		},
	}}

	built, nextLock, err := Build(desc, modules, locked)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if built.Modules[0].Notifies[0].Message.ID != lockedNotify {
		t.Fatalf("existing notify id changed: got %d want %d", built.Modules[0].Notifies[0].Message.ID, lockedNotify)
	}
	if nextLock["BattleFinishedNotify"] != lockedNotify {
		t.Fatalf("lock lost bumped id: %#v", nextLock)
	}
	if _, ok := nextLock["AaaNewNotify"]; !ok {
		t.Fatalf("new message missing from lock: %#v", nextLock)
	}
}

func TestBuildDropsDeletedLockEntriesAndReusesIDs(t *testing.T) {
	goneID := msgid.Compute("GoneNotify", false)
	locked := map[string]uint32{
		"GoneNotify":        goneID,
		"HeartbeatRequest":  msgid.Compute("HeartbeatRequest", true),
		"HeartbeatResponse": msgid.Compute("HeartbeatResponse", false),
	}
	descWithoutGone := NewDescriptorSetForTest(
		msg("transform.HeartbeatRequest", 0, descriptor.MessageKindRequest),
		msg("transform.HeartbeatResponse", 0, descriptor.MessageKindResponse),
	)
	_, nextLock, err := Build(descWithoutGone, []define.Module{{
		Name: "player", Version: define.Version,
		RPCs: []define.RPC{{Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context"}},
	}}, locked)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := nextLock["GoneNotify"]; ok {
		t.Fatalf("deleted message still in lock: %#v", nextLock)
	}

	descWithGone := NewDescriptorSetForTest(
		msg("transform.HeartbeatRequest", 0, descriptor.MessageKindRequest),
		msg("transform.HeartbeatResponse", 0, descriptor.MessageKindResponse),
		msg("transform.GoneNotify", 0, descriptor.MessageKindNotify),
	)
	built, reclaimLock, err := Build(descWithGone, []define.Module{{
		Name: "player", Version: define.Version,
		RPCs: []define.RPC{{Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context"}},
		Notifies: []define.Notify{{Method: "Gone", Message: "transform.GoneNotify", Ctx: "context.Context"}},
	}}, nextLock)
	if err != nil {
		t.Fatal(err)
	}
	if built.Modules[0].Notifies[0].Message.ID != goneID || reclaimLock["GoneNotify"] != goneID {
		t.Fatalf("reused id = %d/%d, want %d", built.Modules[0].Notifies[0].Message.ID, reclaimLock["GoneNotify"], goneID)
	}
}

func TestBuildRejectsDuplicateLockedIDs(t *testing.T) {
	id := msgid.Compute("HeartbeatRequest", true)
	desc := NewDescriptorSetForTest(
		msg("transform.HeartbeatRequest", 0, descriptor.MessageKindRequest),
		msg("transform.HeartbeatResponse", 0, descriptor.MessageKindResponse),
		msg("transform.OtherRequest", 0, descriptor.MessageKindRequest),
		msg("transform.OtherResponse", 0, descriptor.MessageKindResponse),
	)
	_, _, err := Build(desc, []define.Module{{
		Name: "player", Version: define.Version,
		RPCs: []define.RPC{
			{Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context"},
			{Method: "Other", Request: "transform.OtherRequest", Response: "transform.OtherResponse", Ctx: "context.Context"},
		},
	}}, map[string]uint32{
		"HeartbeatRequest": id,
		"OtherRequest":     id,
	})
	if !errors.Is(err, ErrDuplicateMessageID) {
		t.Fatalf("Build() error = %v, want ErrDuplicateMessageID", err)
	}
}

func TestBuildRejectsMissingMessagesKindMismatchAndDuplicateRequest(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		desc := NewDescriptorSetForTest()
		_, _, err := Build(desc, []define.Module{{Name: "player", Version: define.Version, RPCs: []define.RPC{{
			Method: "Heartbeat", Request: "transform.Missing", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
		}}}}, nil)
		if !errors.Is(err, ErrMessageNotFound) {
			t.Fatalf("Build() error = %v, want ErrMessageNotFound", err)
		}
	})

	t.Run("kind", func(t *testing.T) {
		// Name implies Request, but YAML binds it as a notify message.
		desc := NewDescriptorSetForTest(
			msg("transform.HeartbeatRequest", 0, 0),
		)
		_, _, err := Build(desc, []define.Module{{Name: "player", Version: define.Version, Notifies: []define.Notify{{
			Method: "Heartbeat", Message: "transform.HeartbeatRequest", Ctx: "context.Context",
		}}}}, nil)
		if !errors.Is(err, ErrMessageKindMismatch) {
			t.Fatalf("Build() error = %v, want ErrMessageKindMismatch", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		desc := NewDescriptorSetForTest(
			msg("transform.HeartbeatRequest", 1001, descriptor.MessageKindRequest),
			msg("transform.HeartbeatResponse", 1002, descriptor.MessageKindResponse),
		)
		_, _, err := Build(desc, []define.Module{
			{Name: "player", Version: define.Version, RPCs: []define.RPC{{
				Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
			}}},
			{Name: "battle", Version: define.Version, RPCs: []define.RPC{{
				Method: "Heartbeat", Request: "transform.HeartbeatRequest", Response: "transform.HeartbeatResponse", Ctx: "context.Context",
			}}},
		}, nil)
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
		ProtoName:     fullName[strings.LastIndex(fullName, ".")+1:],
		GoImportPath:  "resource/protocol/src/transform",
		GoPackageName: "transformpb",
		GoTypeName:    fullName[strings.LastIndex(fullName, ".")+1:],
	}
}
