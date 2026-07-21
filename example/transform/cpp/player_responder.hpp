#pragma once

#include "protocol_messages.hpp"
#include "protocol_runtime.hpp"

namespace transform::player {

// ResponderHandler implements the player module business logic.
class ResponderHandler {
 public:
  virtual ~ResponderHandler() = default;

  // Heartbeat handles one request; fill response and return true to reply.
  virtual bool Heartbeat(const PacketContext& context,
                             const ::transform::example::HeartbeatRequest& request,
                             ::transform::example::HeartbeatResponse& response) = 0;
};

// Responder decodes player requests, invokes the handler, and replies.
class Responder {
 public:
  Responder(PacketEndpoint& endpoint, ResponderHandler& handler);

  // RegisterHandlers registers one packet callback per request id.
  bool RegisterHandlers();

  // SendBattleFinished sends one one-way notify to the peer connection.
  bool SendBattleFinished(const PacketContext& context,
                         const ::transform::example::BattleFinishedNotify& message);

 private:
  PacketEndpoint* endpoint_;
  ResponderHandler* handler_;
};

}  // namespace transform::player
