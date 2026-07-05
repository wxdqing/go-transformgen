package frame

import (
	"fmt"

	"gitee.com/wxdqing/go-utils/packet"
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
	p := packet.Writer()
	p.WriteUint32(head.MessageID)
	p.WriteUint32(uint32(len(body)))
	p.WriteUint64(head.RequestID)
	p.WriteUint32(head.PacketSeq)
	p.WriteRawBytes(body)
	return p.Data(), p.Return, nil
}

func (PacketFrameCodec) DecodeFrame(data []byte) (Head, []byte, func(), error) {
	if len(data) < packetHeaderSize {
		return Head{}, nil, func() {}, fmt.Errorf("%w: got %d want at least %d", ErrShortFrame, len(data), packetHeaderSize)
	}
	p := packet.Reader(data)
	release := p.Return
	messageID, err := p.ReadUint32()
	if err != nil {
		release()
		return Head{}, nil, func() {}, err
	}
	bodyLen, err := p.ReadUint32()
	if err != nil {
		release()
		return Head{}, nil, func() {}, err
	}
	requestID, err := p.ReadUint64()
	if err != nil {
		release()
		return Head{}, nil, func() {}, err
	}
	packetSeq, err := p.ReadUint32()
	if err != nil {
		release()
		return Head{}, nil, func() {}, err
	}
	body := p.RemainData()
	if uint32(len(body)) != bodyLen {
		release()
		return Head{}, nil, func() {}, fmt.Errorf("%w: got %d want %d", ErrBodyLenMismatch, len(body), bodyLen)
	}
	return Head{
		MessageID: messageID,
		BodyLen:   bodyLen,
		RequestID: requestID,
		PacketSeq: packetSeq,
	}, body, release, nil
}

var _ FrameCodec = PacketFrameCodec{}
