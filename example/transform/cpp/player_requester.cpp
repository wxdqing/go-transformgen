#include "player_requester.hpp"

#include <utility>

namespace transform::player {

Requester::Requester(PacketEndpoint& endpoint) : endpoint_(&endpoint) {}

bool Requester::SendHeartbeat(const PacketContext& context,
                                  const ::transform::example::HeartbeatRequest& request) {
  // Serialize the payload and forward it with the request message id.
  Packet packet;
  packet.message_id = ToMessageId(EMsgToServerType::HeartbeatRequest);
  if (!request.SerializeToString(&packet.payload)) {
    return false;
  }
  return endpoint_->SendPacket(context, packet);
}

bool Requester::OnHeartbeatResponse(HeartbeatResponseCallback callback) {
  if (callback == nullptr) {
    return false;
  }
  // Decode the payload before invoking the typed callback.
  return endpoint_->RegisterPacketCallback(
      ToMessageId(EMsgToClientType::HeartbeatResponse),
      [callback = std::move(callback)](const PacketContext& context,
                                       const Packet& packet) {
        ::transform::example::HeartbeatResponse response;
        if (!response.ParseFromString(packet.payload)) {
          return false;
        }
        return callback(context, response);
      });
}

bool Requester::OnBattleFinished(BattleFinishedCallback callback) {
  if (callback == nullptr) {
    return false;
  }
  // Decode the payload before invoking the typed callback.
  return endpoint_->RegisterPacketCallback(
      ToMessageId(EMsgToClientType::BattleFinishedNotify),
      [callback = std::move(callback)](const PacketContext& context,
                                       const Packet& packet) {
        ::transform::example::BattleFinishedNotify message;
        if (!message.ParseFromString(packet.payload)) {
          return false;
        }
        return callback(context, message);
      });
}

}  // namespace transform::player
