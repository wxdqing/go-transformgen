#pragma once

#include <functional>

#include "protocol_messages.hpp"
#include "protocol_runtime.hpp"

namespace transform::battle {

// Requester sends battle requests and receives responses and notifies.
class Requester {
 public:
  // StartBattleResponseCallback consumes one decoded StartBattle response.
  using StartBattleResponseCallback =
      std::function<bool(const PacketContext&, const ::transform::example::StartBattleResponse&)>;
  // BattleStateCallback consumes one decoded BattleState notify.
  using BattleStateCallback =
      std::function<bool(const PacketContext&, const ::transform::example::BattleStateNotify&)>;

  explicit Requester(PacketEndpoint& endpoint);

  // SendStartBattle serializes and sends one request; the caller owns
  // request_id correlation through context.
  bool SendStartBattle(const PacketContext& context,
                         const ::transform::example::StartBattleRequest& request);

  // OnStartBattleResponse registers decode-and-dispatch for the response id.
  bool OnStartBattleResponse(StartBattleResponseCallback callback);

  // OnBattleState registers decode-and-dispatch for the notify id.
  bool OnBattleState(BattleStateCallback callback);

 private:
  PacketEndpoint* endpoint_;
};

}  // namespace transform::battle
