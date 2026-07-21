#pragma once

#include <functional>

#include "protocol_messages.hpp"
#include "protocol_runtime.hpp"

namespace transform::player {

// Requester sends player requests and receives responses and notifies.
class Requester {
 public:
  // HeartbeatResponseCallback consumes one decoded Heartbeat response.
  using HeartbeatResponseCallback =
      std::function<bool(const PacketContext&, const ::transform::example::HeartbeatResponse&)>;
  // BattleFinishedCallback consumes one decoded BattleFinished notify.
  using BattleFinishedCallback =
      std::function<bool(const PacketContext&, const ::transform::example::BattleFinishedNotify&)>;

  explicit Requester(PacketEndpoint& endpoint);

  // SendHeartbeat serializes and sends one request; the caller owns
  // request_id correlation through context.
  bool SendHeartbeat(const PacketContext& context,
                         const ::transform::example::HeartbeatRequest& request);

  // OnHeartbeatResponse registers decode-and-dispatch for the response id.
  bool OnHeartbeatResponse(HeartbeatResponseCallback callback);

  // OnBattleFinished registers decode-and-dispatch for the notify id.
  bool OnBattleFinished(BattleFinishedCallback callback);

 private:
  PacketEndpoint* endpoint_;
};

}  // namespace transform::player
