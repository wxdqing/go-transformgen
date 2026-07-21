package cpptarget

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// protobufStub is a minimal google::protobuf::Message replacement so the
// generated code can be syntax-checked without the real protobuf runtime.
const protobufStub = `#pragma once
#include <string>
namespace google::protobuf {
class Message {
 public:
  virtual ~Message() = default;
  bool SerializeToString(std::string* output) const {
    output->clear();
    return true;
  }
  bool ParseFromString(const std::string&) { return true; }
};
}  // namespace google::protobuf
`

// heartbeatStub mirrors the protoc output referenced by testModel.
const heartbeatStub = `#pragma once
#include <google/protobuf/message.h>
namespace transform {
class HeartbeatRequest : public ::google::protobuf::Message {};
class HeartbeatResponse : public ::google::protobuf::Message {};
}  // namespace transform
`

// battleStub mirrors the protoc output referenced by testModel.
const battleStub = `#pragma once
#include <google/protobuf/message.h>
namespace transform {
class BattleFinishedNotify : public ::google::protobuf::Message {};
}  // namespace transform
`

// driverStub instantiates the generated classes and templates.
const driverStub = `#include "player_requester.hpp"
#include "player_responder.hpp"

namespace {

class FakeEndpoint final : public transform::PacketEndpoint {
 public:
  bool RegisterPacketCallback(std::uint32_t, transform::PacketCallback) override {
    return true;
  }
  bool SendPacket(const transform::PacketContext&, const transform::Packet&) override {
    return true;
  }
};

class FakeHandler final : public transform::player::ResponderHandler {
 public:
  bool Heartbeat(const transform::PacketContext&,
                 const ::transform::HeartbeatRequest&,
                 ::transform::HeartbeatResponse&) override {
    return true;
  }
};

}  // namespace

int main() {
  FakeEndpoint endpoint;
  FakeHandler handler;
  transform::player::Requester requester(endpoint);
  transform::player::Responder responder(endpoint, handler);
  static_assert(transform::MessageIdOf<::transform::HeartbeatRequest> == 200012345u);
  static_assert(transform::ToMessageId(transform::EMsgToServerType::HeartbeatRequest) == 200012345u);
  (void)requester.OnHeartbeatResponse(nullptr);
  (void)responder.RegisterHandlers();
  (void)transform::CreateMessage(200012345u);
  (void)transform::FindMessageMeta(200012345u);
  return 0;
}
`

// TestGeneratedCppSyntax verifies the generated output is valid C++23.
func TestGeneratedCppSyntax(t *testing.T) {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		t.Skip("g++ not found; skipping C++ syntax check")
	}

	// Render the sample model into a temporary source tree.
	files, err := Render(testModel(), Options{Namespace: "transform"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	dir := t.TempDir()
	for _, file := range files {
		writeFile(t, filepath.Join(dir, file.Path), file.Content)
	}

	// Provide stub protobuf headers and a driver that uses the API.
	writeFile(t, filepath.Join(dir, "google", "protobuf", "message.h"), []byte(protobufStub))
	writeFile(t, filepath.Join(dir, "heartbeat.pb.h"), []byte(heartbeatStub))
	writeFile(t, filepath.Join(dir, "battle.pb.h"), []byte(battleStub))
	writeFile(t, filepath.Join(dir, "driver.cpp"), []byte(driverStub))

	// Syntax-check every translation unit under C++23.
	for _, source := range []string{
		"protocol_messages.cpp",
		"player_requester.cpp",
		"player_responder.cpp",
		"driver.cpp",
	} {
		cmd := exec.Command(compiler, "-std=c++23", "-fsyntax-only", "-Wall", "-Wextra", "-Werror", "-I", dir, source)
		cmd.Dir = dir
		if output, runErr := cmd.CombinedOutput(); runErr != nil {
			t.Fatalf("g++ %s failed: %v\n%s", source, runErr, output)
		}
	}
}

// writeFile writes one file creating parent directories.
func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
