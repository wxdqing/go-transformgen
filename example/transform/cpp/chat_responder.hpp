#pragma once

#include "protocol_messages.hpp"
#include "protocol_runtime.hpp"

namespace transform::chat {

// ResponderHandler implements the chat module business logic.
class ResponderHandler {
 public:
  virtual ~ResponderHandler() = default;

  // SendChat handles one request; fill response and return true to reply.
  virtual bool SendChat(const PacketContext& context,
                             const ::transform::example::SendChatRequest& request,
                             ::transform::example::SendChatResponse& response) = 0;
};

// Responder decodes chat requests, invokes the handler, and replies.
class Responder {
 public:
  Responder(PacketEndpoint& endpoint, ResponderHandler& handler);

  // RegisterHandlers registers one packet callback per request id.
  bool RegisterHandlers();

  // SendChatMessage sends one one-way notify to the peer connection.
  bool SendChatMessage(const PacketContext& context,
                         const ::transform::example::ChatMessageNotify& message);

 private:
  PacketEndpoint* endpoint_;
  ResponderHandler* handler_;
};

}  // namespace transform::chat
