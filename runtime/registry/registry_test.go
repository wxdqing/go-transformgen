package registry

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRegistryParsesRegisteredRequestResponseAndNotify(t *testing.T) {
	reg := New()
	if err := reg.RegisterRequest(MessageMeta{ID: 1001, Kind: MessageKindRequest, FullName: "test.Request"}, func() proto.Message {
		return &wrapperspb.StringValue{}
	}); err != nil {
		t.Fatalf("RegisterRequest() error = %v", err)
	}
	if err := reg.RegisterResponse(MessageMeta{ID: 1002, Kind: MessageKindResponse, FullName: "test.Response"}, func() proto.Message {
		return &wrapperspb.Int64Value{}
	}); err != nil {
		t.Fatalf("RegisterResponse() error = %v", err)
	}
	if err := reg.RegisterNotify(MessageMeta{ID: 2001, Kind: MessageKindNotify, FullName: "test.Notify"}, func() proto.Message {
		return &emptypb.Empty{}
	}); err != nil {
		t.Fatalf("RegisterNotify() error = %v", err)
	}

	reqPayload, err := proto.Marshal(wrapperspb.String("hello"))
	if err != nil {
		t.Fatal(err)
	}
	req, err := reg.ParseRequest(1001, reqPayload)
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}
	if got := req.(*wrapperspb.StringValue).GetValue(); got != "hello" {
		t.Fatalf("request value = %q, want hello", got)
	}

	respPayload, err := proto.Marshal(wrapperspb.Int64(42))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := reg.ParseResponse(1002, respPayload)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if got := resp.(*wrapperspb.Int64Value).GetValue(); got != 42 {
		t.Fatalf("response value = %d, want 42", got)
	}

	notifyPayload, err := proto.Marshal(&emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.ParseNotify(2001, notifyPayload); err != nil {
		t.Fatalf("ParseNotify() error = %v", err)
	}
}

func TestRegistryRejectsDuplicateMessageIDAndWrongKind(t *testing.T) {
	reg := New()
	if err := reg.RegisterRequest(MessageMeta{ID: 1001, Kind: MessageKindRequest, FullName: "test.Request"}, func() proto.Message {
		return &wrapperspb.StringValue{}
	}); err != nil {
		t.Fatalf("RegisterRequest() error = %v", err)
	}
	if err := reg.RegisterResponse(MessageMeta{ID: 1001, Kind: MessageKindResponse, FullName: "test.Response"}, func() proto.Message {
		return &wrapperspb.Int64Value{}
	}); !errors.Is(err, ErrDuplicateMessageID) {
		t.Fatalf("RegisterResponse() error = %v, want ErrDuplicateMessageID", err)
	}

	payload, err := proto.Marshal(wrapperspb.String("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.ParseResponse(1001, payload); !errors.Is(err, ErrMessageKindMismatch) {
		t.Fatalf("ParseResponse() error = %v, want ErrMessageKindMismatch", err)
	}
	if _, err := reg.ParseRequest(9999, payload); !errors.Is(err, ErrUnknownMessageID) {
		t.Fatalf("ParseRequest() error = %v, want ErrUnknownMessageID", err)
	}
}

func TestRegistryDispatchesRequestAndNotifyHandlers(t *testing.T) {
	reg := New()
	if err := reg.RegisterRequest(MessageMeta{ID: 1001, Kind: MessageKindRequest, FullName: "test.Request"}, func() proto.Message {
		return &wrapperspb.StringValue{}
	}); err != nil {
		t.Fatalf("RegisterRequest() error = %v", err)
	}
	if err := reg.RegisterResponse(MessageMeta{ID: 1002, Kind: MessageKindResponse, FullName: "test.Response"}, func() proto.Message {
		return &wrapperspb.StringValue{}
	}); err != nil {
		t.Fatalf("RegisterResponse() error = %v", err)
	}
	if err := reg.RegisterNotify(MessageMeta{ID: 2001, Kind: MessageKindNotify, FullName: "test.Notify"}, func() proto.Message {
		return &wrapperspb.StringValue{}
	}); err != nil {
		t.Fatalf("RegisterNotify() error = %v", err)
	}

	if err := reg.RegisterRequestHandler("player", 1001, 1002, func(ctx any, req proto.Message) (proto.Message, error) {
		prefix, ok := ctx.(string)
		if !ok {
			return nil, ErrInvalidContextType
		}
		value, ok := req.(*wrapperspb.StringValue)
		if !ok {
			return nil, ErrInvalidMessageType
		}
		return wrapperspb.String(prefix + value.GetValue()), nil
	}); err != nil {
		t.Fatalf("RegisterRequestHandler() error = %v", err)
	}

	reqPayload, err := proto.Marshal(wrapperspb.String("pong"))
	if err != nil {
		t.Fatal(err)
	}
	resp, respID, err := reg.DispatchRequest("ping:", 1001, reqPayload)
	if err != nil {
		t.Fatalf("DispatchRequest() error = %v", err)
	}
	if respID != 1002 {
		t.Fatalf("response id = %d, want 1002", respID)
	}
	if got := resp.(*wrapperspb.StringValue).GetValue(); got != "ping:pong" {
		t.Fatalf("response value = %q, want ping:pong", got)
	}

	seen := ""
	if err := reg.RegisterNotifyHandler("player", 2001, func(ctx any, msg proto.Message) error {
		_ = ctx.(context.Context)
		seen = msg.(*wrapperspb.StringValue).GetValue()
		return nil
	}); err != nil {
		t.Fatalf("RegisterNotifyHandler() error = %v", err)
	}
	notifyPayload, err := proto.Marshal(wrapperspb.String("event"))
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.DispatchNotify(context.Background(), 2001, notifyPayload); err != nil {
		t.Fatalf("DispatchNotify() error = %v", err)
	}
	if seen != "event" {
		t.Fatalf("notify seen = %q, want event", seen)
	}
	if err := reg.RegisterNotifyHandler("player", 2001, func(any, proto.Message) error { return nil }); !errors.Is(err, ErrDuplicateHandler) {
		t.Fatalf("duplicate notify handler error = %v, want ErrDuplicateHandler", err)
	}
}
