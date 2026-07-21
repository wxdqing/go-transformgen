#pragma once

#include <functional>

#include "protocol_messages.hpp"
#include "protocol_runtime.hpp"

namespace transform::chat {

// Requester sends chat requests and receives responses and notifies.
class Requester {
 public:
  // SendChatResponseCallback consumes one decoded SendChat response.
  using SendChatResponseCallback =
      std::function<bool(const PacketContext&, const ::transform::example::SendChatResponse&)>;
  // ChatMessageCallback consumes one decoded ChatMessage notify.
  using ChatMessageCallback =
      std::function<bool(const PacketContext&, const ::transform::example::ChatMessageNotify&)>;

  explicit Requester(PacketEndpoint& endpoint);

  // SendSendChat serializes and sends one request; the caller owns
  // request_id correlation through context.
  bool SendSendChat(const PacketContext& context,
                         const ::transform::example::SendChatRequest& request);

  // OnSendChatResponse registers decode-and-dispatch for the response id.
  bool OnSendChatResponse(SendChatResponseCallback callback);

  // OnChatMessage registers decode-and-dispatch for the notify id.
  bool OnChatMessage(ChatMessageCallback callback);

 private:
  PacketEndpoint* endpoint_;
};

}  // namespace transform::chat
