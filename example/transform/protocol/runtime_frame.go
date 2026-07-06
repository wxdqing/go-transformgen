package protocolpb

import (
	"encoding/binary"
	"fmt"
)

const packetHeaderSize = 20

type Head struct {
	MessageID uint32
	BodyLen   uint32
	RequestID uint64
	PacketSeq uint32
}

type FrameCodec interface {
	EncodeFrame(head Head, body []byte) ([]byte, func(), error)
	DecodeFrame(data []byte) (Head, []byte, func(), error)
}

type PacketFrameCodec struct{}

func (PacketFrameCodec) EncodeFrame(head Head, body []byte) ([]byte, func(), error) {
	out := make([]byte, packetHeaderSize+len(body))
	binary.BigEndian.PutUint32(out[0:4], head.MessageID)
	binary.BigEndian.PutUint32(out[4:8], uint32(len(body)))
	binary.BigEndian.PutUint64(out[8:16], head.RequestID)
	binary.BigEndian.PutUint32(out[16:20], head.PacketSeq)
	copy(out[20:], body)
	return out, func() {}, nil
}

func (PacketFrameCodec) DecodeFrame(data []byte) (Head, []byte, func(), error) {
	if len(data) < packetHeaderSize {
		return Head{}, nil, func() {}, fmt.Errorf("%w: got %d want at least %d", ErrShortFrame, len(data), packetHeaderSize)
	}
	bodyLen := binary.BigEndian.Uint32(data[4:8])
	body := data[20:]
	if uint32(len(body)) != bodyLen {
		return Head{}, nil, func() {}, fmt.Errorf("%w: got %d want %d", ErrBodyLenMismatch, len(body), bodyLen)
	}
	return Head{
		MessageID: binary.BigEndian.Uint32(data[0:4]),
		BodyLen:   bodyLen,
		RequestID: binary.BigEndian.Uint64(data[8:16]),
		PacketSeq: binary.BigEndian.Uint32(data[16:20]),
	}, body, func() {}, nil
}

var _ FrameCodec = PacketFrameCodec{}
