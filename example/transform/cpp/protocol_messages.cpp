#include "protocol_messages.hpp"

namespace transform {
namespace {

// kMessageMetas is sorted by message id (matches generation order).
constexpr MessageMeta kMessageMetas[] = {
    {114012388u, MessageKind::Notify, "transform.example.BattleStateNotify"},
    {124399461u, MessageKind::Response, "transform.example.SendChatResponse"},
    {143096507u, MessageKind::Notify, "transform.example.BattleFinishedNotify"},
    {168595187u, MessageKind::Response, "transform.example.HeartbeatResponse"},
    {170889542u, MessageKind::Notify, "transform.example.ChatMessageNotify"},
    {171396577u, MessageKind::Response, "transform.example.StartBattleResponse"},
    {234959079u, MessageKind::Request, "transform.example.StartBattleRequest"},
    {235223567u, MessageKind::Request, "transform.example.SendChatRequest"},
    {259926425u, MessageKind::Request, "transform.example.HeartbeatRequest"},
};

}  // namespace

const MessageMeta* FindMessageMeta(std::uint32_t message_id) noexcept {
  // The table is small; scan linearly to keep the code obvious.
  for (const MessageMeta& meta : kMessageMetas) {
    if (meta.message_id == message_id) {
      return &meta;
    }
  }
  return nullptr;
}

std::unique_ptr<google::protobuf::Message> CreateMessage(std::uint32_t message_id) {
  // Instantiate the official protoc C++ type for a known id.
  switch (message_id) {
    case 114012388u:
      return std::make_unique<::transform::example::BattleStateNotify>();
    case 124399461u:
      return std::make_unique<::transform::example::SendChatResponse>();
    case 143096507u:
      return std::make_unique<::transform::example::BattleFinishedNotify>();
    case 168595187u:
      return std::make_unique<::transform::example::HeartbeatResponse>();
    case 170889542u:
      return std::make_unique<::transform::example::ChatMessageNotify>();
    case 171396577u:
      return std::make_unique<::transform::example::StartBattleResponse>();
    case 234959079u:
      return std::make_unique<::transform::example::StartBattleRequest>();
    case 235223567u:
      return std::make_unique<::transform::example::SendChatRequest>();
    case 259926425u:
      return std::make_unique<::transform::example::HeartbeatRequest>();
    default:
      return nullptr;
  }
}

}  // namespace transform
