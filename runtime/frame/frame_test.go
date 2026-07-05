package frame

import (
	"errors"
	"testing"
)

func TestPacketFrameCodecRoundTrip(t *testing.T) {
	codec := PacketFrameCodec{}
	body := []byte("payload")
	raw, release, err := codec.EncodeFrame(Head{
		MessageID: 1001,
		RequestID: 55,
		PacketSeq: 7,
	}, body)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	defer release()

	head, decodedBody, decodedRelease, err := codec.DecodeFrame(raw)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	defer decodedRelease()

	if head.MessageID != 1001 || head.BodyLen != uint32(len(body)) || head.RequestID != 55 || head.PacketSeq != 7 {
		t.Fatalf("head = %+v", head)
	}
	if string(decodedBody) != "payload" {
		t.Fatalf("body = %q, want payload", string(decodedBody))
	}
}

func TestPacketFrameCodecRejectsBodyLengthMismatch(t *testing.T) {
	codec := PacketFrameCodec{}
	raw, release, err := codec.EncodeFrame(Head{MessageID: 1001}, []byte("payload"))
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	defer release()

	raw[7] = raw[7] + 1
	_, _, decodedRelease, err := codec.DecodeFrame(raw)
	defer decodedRelease()
	if !errors.Is(err, ErrBodyLenMismatch) {
		t.Fatalf("DecodeFrame() error = %v, want ErrBodyLenMismatch", err)
	}
}

func TestPacketFrameCodecRejectsShortFrame(t *testing.T) {
	codec := PacketFrameCodec{}
	_, _, release, err := codec.DecodeFrame([]byte{1, 2, 3})
	defer release()
	if !errors.Is(err, ErrShortFrame) {
		t.Fatalf("DecodeFrame() error = %v, want ErrShortFrame", err)
	}
}
