#pragma once

#include <cstdint>
#include <memory>

#include <google/protobuf/message.h>

#include "battle.pb.h"
#include "chat.pb.h"
#include "heartbeat.pb.h"

namespace transform {

// EMsgToServerType enumerates client-to-server request message ids.
enum class EMsgToServerType : std::uint32_t {
  StartBattleRequest = 234959079,
  SendChatRequest = 235223567,
  HeartbeatRequest = 259926425,
};

// EMsgToClientType enumerates server-to-client response and notify message ids.
enum class EMsgToClientType : std::uint32_t {
  BattleStateNotify = 114012388,
  SendChatResponse = 124399461,
  BattleFinishedNotify = 143096507,
  HeartbeatResponse = 168595187,
  ChatMessageNotify = 170889542,
  StartBattleResponse = 171396577,
};

// ToMessageId converts a direction enum to its wire message id.
constexpr std::uint32_t ToMessageId(EMsgToServerType type) noexcept {
  return static_cast<std::uint32_t>(type);
}

// ToMessageId converts a direction enum to its wire message id.
constexpr std::uint32_t ToMessageId(EMsgToClientType type) noexcept {
  return static_cast<std::uint32_t>(type);
}

// MessageKind classifies protocol messages by their transport role.
enum class MessageKind : std::uint8_t {
  Unknown = 0,
  Request = 1,
  Response = 2,
  Notify = 3,
};

// MessageMeta describes one protocol message.
struct MessageMeta {
  std::uint32_t message_id{};
  MessageKind kind{MessageKind::Unknown};
  const char* full_name{""};
};

// FindMessageMeta returns metadata for a message id, or nullptr when unknown.
const MessageMeta* FindMessageMeta(std::uint32_t message_id) noexcept;

// CreateMessage builds an empty protobuf instance for a message id, or nullptr when unknown.
std::unique_ptr<google::protobuf::Message> CreateMessage(std::uint32_t message_id);

// MessageId maps a protobuf type to its wire message id at compile time.
template <typename Message>
struct MessageId;


template <>
struct MessageId<::transform::example::BattleStateNotify> {
  static constexpr std::uint32_t value = 114012388;
};

template <>
struct MessageId<::transform::example::SendChatResponse> {
  static constexpr std::uint32_t value = 124399461;
};

template <>
struct MessageId<::transform::example::BattleFinishedNotify> {
  static constexpr std::uint32_t value = 143096507;
};

template <>
struct MessageId<::transform::example::HeartbeatResponse> {
  static constexpr std::uint32_t value = 168595187;
};

template <>
struct MessageId<::transform::example::ChatMessageNotify> {
  static constexpr std::uint32_t value = 170889542;
};

template <>
struct MessageId<::transform::example::StartBattleResponse> {
  static constexpr std::uint32_t value = 171396577;
};

template <>
struct MessageId<::transform::example::StartBattleRequest> {
  static constexpr std::uint32_t value = 234959079;
};

template <>
struct MessageId<::transform::example::SendChatRequest> {
  static constexpr std::uint32_t value = 235223567;
};

template <>
struct MessageId<::transform::example::HeartbeatRequest> {
  static constexpr std::uint32_t value = 259926425;
};

// MessageIdOf is the variable-template shorthand for MessageId<Message>::value.
template <typename Message>
inline constexpr std::uint32_t MessageIdOf = MessageId<Message>::value;

}  // namespace transform
