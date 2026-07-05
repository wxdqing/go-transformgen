package registry

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

type MessageKind uint8

const (
	MessageKindRequest  MessageKind = 1
	MessageKindResponse MessageKind = 2
	MessageKindNotify   MessageKind = 3
)

type MessageMeta struct {
	ID       uint32
	Kind     MessageKind
	FullName string
}

type MessageFactory func() proto.Message

type RequestHandler func(ctx any, req proto.Message) (proto.Message, error)
type NotifyHandler func(ctx any, msg proto.Message) error

type MessageRegistry interface {
	RegisterRequest(meta MessageMeta, newMessage MessageFactory) error
	RegisterResponse(meta MessageMeta, newMessage MessageFactory) error
	RegisterNotify(meta MessageMeta, newMessage MessageFactory) error

	ParseRequest(messageID uint32, payload []byte) (proto.Message, error)
	ParseResponse(messageID uint32, payload []byte) (proto.Message, error)
	ParseNotify(messageID uint32, payload []byte) (proto.Message, error)
	ParseMessage(messageID uint32, payload []byte) (proto.Message, MessageMeta, error)
}

type HandlerRegistry interface {
	RegisterRequestHandler(modelName string, requestID uint32, responseID uint32, handler RequestHandler) error
	RegisterNotifyHandler(modelName string, notifyID uint32, handler NotifyHandler) error

	DispatchRequest(ctx any, messageID uint32, payload []byte) (proto.Message, uint32, error)
	DispatchNotify(ctx any, messageID uint32, payload []byte) error
}

type Registry interface {
	MessageRegistry
	HandlerRegistry
}

type registeredMessage struct {
	meta       MessageMeta
	newMessage MessageFactory
}

type requestRoute struct {
	modelName  string
	responseID uint32
	handler    RequestHandler
}

type notifyRoute struct {
	modelName string
	handler   NotifyHandler
}

type Default struct {
	messages map[uint32]registeredMessage
	requests map[uint32]requestRoute
	notifies map[uint32]notifyRoute
}

func New() *Default {
	return &Default{
		messages: make(map[uint32]registeredMessage),
		requests: make(map[uint32]requestRoute),
		notifies: make(map[uint32]notifyRoute),
	}
}

func DefaultRegistry() *Default {
	return New()
}

func (r *Default) RegisterRequest(meta MessageMeta, newMessage MessageFactory) error {
	meta.Kind = MessageKindRequest
	return r.registerMessage(meta, newMessage)
}

func (r *Default) RegisterResponse(meta MessageMeta, newMessage MessageFactory) error {
	meta.Kind = MessageKindResponse
	return r.registerMessage(meta, newMessage)
}

func (r *Default) RegisterNotify(meta MessageMeta, newMessage MessageFactory) error {
	meta.Kind = MessageKindNotify
	return r.registerMessage(meta, newMessage)
}

func (r *Default) registerMessage(meta MessageMeta, newMessage MessageFactory) error {
	if r == nil {
		return fmt.Errorf("%w: nil registry", ErrUnknownMessageID)
	}
	if newMessage == nil {
		return fmt.Errorf("%w: nil factory for %d", ErrInvalidMessageType, meta.ID)
	}
	if _, exists := r.messages[meta.ID]; exists {
		return fmt.Errorf("%w: %d", ErrDuplicateMessageID, meta.ID)
	}
	r.messages[meta.ID] = registeredMessage{meta: meta, newMessage: newMessage}
	return nil
}

func (r *Default) ParseRequest(messageID uint32, payload []byte) (proto.Message, error) {
	return r.parseKind(messageID, payload, MessageKindRequest)
}

func (r *Default) ParseResponse(messageID uint32, payload []byte) (proto.Message, error) {
	return r.parseKind(messageID, payload, MessageKindResponse)
}

func (r *Default) ParseNotify(messageID uint32, payload []byte) (proto.Message, error) {
	return r.parseKind(messageID, payload, MessageKindNotify)
}

func (r *Default) ParseMessage(messageID uint32, payload []byte) (proto.Message, MessageMeta, error) {
	msg, meta, err := r.parse(messageID, payload)
	return msg, meta, err
}

func (r *Default) parseKind(messageID uint32, payload []byte, kind MessageKind) (proto.Message, error) {
	msg, meta, err := r.parse(messageID, payload)
	if err != nil {
		return nil, err
	}
	if meta.Kind != kind {
		return nil, fmt.Errorf("%w: message %d kind %d want %d", ErrMessageKindMismatch, messageID, meta.Kind, kind)
	}
	return msg, nil
}

func (r *Default) parse(messageID uint32, payload []byte) (proto.Message, MessageMeta, error) {
	if r == nil {
		return nil, MessageMeta{}, fmt.Errorf("%w: nil registry", ErrUnknownMessageID)
	}
	registered, ok := r.messages[messageID]
	if !ok {
		return nil, MessageMeta{}, fmt.Errorf("%w: %d", ErrUnknownMessageID, messageID)
	}
	msg := registered.newMessage()
	if msg == nil {
		return nil, MessageMeta{}, fmt.Errorf("%w: nil factory result for %d", ErrInvalidMessageType, messageID)
	}
	if err := proto.Unmarshal(payload, msg); err != nil {
		return nil, MessageMeta{}, err
	}
	return msg, registered.meta, nil
}

func (r *Default) RegisterRequestHandler(modelName string, requestID uint32, responseID uint32, handler RequestHandler) error {
	if handler == nil {
		return fmt.Errorf("%w: nil request handler for %d", ErrInvalidMessageType, requestID)
	}
	if _, exists := r.requests[requestID]; exists {
		return fmt.Errorf("%w: request %d", ErrDuplicateHandler, requestID)
	}
	r.requests[requestID] = requestRoute{modelName: modelName, responseID: responseID, handler: handler}
	return nil
}

func (r *Default) RegisterNotifyHandler(modelName string, notifyID uint32, handler NotifyHandler) error {
	if handler == nil {
		return fmt.Errorf("%w: nil notify handler for %d", ErrInvalidMessageType, notifyID)
	}
	if _, exists := r.notifies[notifyID]; exists {
		return fmt.Errorf("%w: notify %d", ErrDuplicateHandler, notifyID)
	}
	r.notifies[notifyID] = notifyRoute{modelName: modelName, handler: handler}
	return nil
}

func (r *Default) DispatchRequest(ctx any, messageID uint32, payload []byte) (proto.Message, uint32, error) {
	route, ok := r.requests[messageID]
	if !ok {
		return nil, 0, fmt.Errorf("%w: request %d", ErrHandlerNotFound, messageID)
	}
	req, err := r.ParseRequest(messageID, payload)
	if err != nil {
		return nil, 0, err
	}
	resp, err := route.handler(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	return resp, route.responseID, nil
}

func (r *Default) DispatchNotify(ctx any, messageID uint32, payload []byte) error {
	route, ok := r.notifies[messageID]
	if !ok {
		return fmt.Errorf("%w: notify %d", ErrHandlerNotFound, messageID)
	}
	msg, err := r.ParseNotify(messageID, payload)
	if err != nil {
		return err
	}
	return route.handler(ctx, msg)
}

var _ Registry = (*Default)(nil)
