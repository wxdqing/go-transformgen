package descriptor

import (
	"fmt"
	"os"
	"strings"

	"github.com/wxdqing/go-transformgen/proto/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type MessageKind uint8

const (
	MessageKindRequest  MessageKind = 1
	MessageKindResponse MessageKind = 2
	MessageKindNotify   MessageKind = 3
)

type Message struct {
	ID            uint32
	Kind          MessageKind
	FullName      string
	GoImportPath  string
	GoPackageName string
	GoTypeName    string
}

type Set struct {
	messages map[string]Message
	ids      map[uint32]Message
}

func NewSet(messages ...Message) *Set {
	out := &Set{
		messages: make(map[string]Message),
		ids:      make(map[uint32]Message),
	}
	for _, msg := range messages {
		out.messages[msg.FullName] = msg
		out.ids[msg.ID] = msg
	}
	return out
}

func Load(path string) (*Set, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var files descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(raw, &files); err != nil {
		return nil, err
	}
	out := &Set{
		messages: make(map[string]Message),
		ids:      make(map[uint32]Message),
	}
	for _, file := range files.GetFile() {
		goImport, goPackage := splitGoPackage(file.GetOptions().GetGoPackage())
		idRange := readMessageIDRange(file.GetOptions())
		for _, msg := range file.GetMessageType() {
			message, ok := readMessage(file.GetPackage(), goImport, goPackage, msg)
			if !ok {
				continue
			}
			if !idRange.contains(message.ID) {
				return nil, fmt.Errorf("%w: %s message_id %d not in [%d, %d]", ErrMessageIDOutOfRange, message.FullName, message.ID, idRange.min, idRange.max)
			}
			out.messages[message.FullName] = message
			out.ids[message.ID] = message
		}
	}
	return out, nil
}

func (s *Set) Message(fullName string) (Message, bool) {
	if s == nil {
		return Message{}, false
	}
	msg, ok := s.messages[fullName]
	return msg, ok
}

func (s *Set) MessageID(id uint32) (Message, bool) {
	if s == nil {
		return Message{}, false
	}
	msg, ok := s.ids[id]
	return msg, ok
}

func (s *Set) Messages() []Message {
	if s == nil {
		return nil
	}
	out := make([]Message, 0, len(s.messages))
	for _, msg := range s.messages {
		out = append(out, msg)
	}
	return out
}

type messageIDRange struct {
	has bool
	min uint32
	max uint32
}

func (r messageIDRange) contains(id uint32) bool {
	if !r.has {
		return true
	}
	return id >= r.min && id <= r.max
}

func readMessageIDRange(opts *descriptorpb.FileOptions) messageIDRange {
	if opts == nil || !proto.HasExtension(opts, options.E_MessageIdMin) || !proto.HasExtension(opts, options.E_MessageIdMax) {
		return messageIDRange{}
	}
	min, minOK := proto.GetExtension(opts, options.E_MessageIdMin).(uint32)
	max, maxOK := proto.GetExtension(opts, options.E_MessageIdMax).(uint32)
	if !minOK || !maxOK {
		return messageIDRange{}
	}
	return messageIDRange{has: true, min: min, max: max}
}

func readMessage(pkg string, goImport string, goPackage string, msg *descriptorpb.DescriptorProto) (Message, bool) {
	opts := msg.GetOptions()
	if opts == nil || !proto.HasExtension(opts, options.E_MessageId) || !proto.HasExtension(opts, options.E_MessageKind) {
		return Message{}, false
	}
	id, ok := proto.GetExtension(opts, options.E_MessageId).(uint32)
	if !ok {
		return Message{}, false
	}
	kindValue, ok := proto.GetExtension(opts, options.E_MessageKind).(options.MessageKind)
	if !ok {
		return Message{}, false
	}
	fullName := msg.GetName()
	if pkg != "" {
		fullName = pkg + "." + fullName
	}
	return Message{
		ID:            id,
		Kind:          convertKind(kindValue),
		FullName:      fullName,
		GoImportPath:  goImport,
		GoPackageName: goPackage,
		GoTypeName:    msg.GetName(),
	}, true
}

func convertKind(kind options.MessageKind) MessageKind {
	switch kind {
	case options.MessageKind_MESSAGE_KIND_REQUEST:
		return MessageKindRequest
	case options.MessageKind_MESSAGE_KIND_RESPONSE:
		return MessageKindResponse
	case options.MessageKind_MESSAGE_KIND_NOTIFY:
		return MessageKindNotify
	default:
		return 0
	}
}

func splitGoPackage(value string) (string, string) {
	if value == "" {
		return "", ""
	}
	parts := strings.Split(value, ";")
	if len(parts) == 1 {
		pathParts := strings.Split(parts[0], "/")
		return parts[0], pathParts[len(pathParts)-1]
	}
	return parts[0], parts[1]
}
