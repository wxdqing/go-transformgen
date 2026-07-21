#pragma once

#include "protocol_messages.hpp"
#include "protocol_runtime.hpp"

namespace transform::battle {

// ResponderHandler implements the battle module business logic.
class ResponderHandler {
 public:
  virtual ~ResponderHandler() = default;

  // StartBattle handles one request; fill response and return true to reply.
  virtual bool StartBattle(const PacketContext& context,
                             const ::transform::example::StartBattleRequest& request,
                             ::transform::example::StartBattleResponse& response) = 0;
};

// Responder decodes battle requests, invokes the handler, and replies.
class Responder {
 public:
  Responder(PacketEndpoint& endpoint, ResponderHandler& handler);

  // RegisterHandlers registers one packet callback per request id.
  bool RegisterHandlers();

  // SendBattleState sends one one-way notify to the peer connection.
  bool SendBattleState(const PacketContext& context,
                         const ::transform::example::BattleStateNotify& message);

 private:
  PacketEndpoint* endpoint_;
  ResponderHandler* handler_;
};

}  // namespace transform::battle
