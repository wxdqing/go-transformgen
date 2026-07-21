#include "battle_requester.hpp"

#include <utility>

namespace transform::battle {

Requester::Requester(PacketEndpoint& endpoint) : endpoint_(&endpoint) {}

bool Requester::SendStartBattle(const PacketContext& context,
                                  const ::transform::example::StartBattleRequest& request) {
  // Serialize the payload and forward it with the request message id.
  Packet packet;
  packet.message_id = ToMessageId(EMsgToServerType::StartBattleRequest);
  if (!request.SerializeToString(&packet.payload)) {
    return false;
  }
  return endpoint_->SendPacket(context, packet);
}

bool Requester::OnStartBattleResponse(StartBattleResponseCallback callback) {
  if (callback == nullptr) {
    return false;
  }
  // Decode the payload before invoking the typed callback.
  return endpoint_->RegisterPacketCallback(
      ToMessageId(EMsgToClientType::StartBattleResponse),
      [callback = std::move(callback)](const PacketContext& context,
                                       const Packet& packet) {
        ::transform::example::StartBattleResponse response;
        if (!response.ParseFromString(packet.payload)) {
          return false;
        }
        return callback(context, response);
      });
}

bool Requester::OnBattleState(BattleStateCallback callback) {
  if (callback == nullptr) {
    return false;
  }
  // Decode the payload before invoking the typed callback.
  return endpoint_->RegisterPacketCallback(
      ToMessageId(EMsgToClientType::BattleStateNotify),
      [callback = std::move(callback)](const PacketContext& context,
                                       const Packet& packet) {
        ::transform::example::BattleStateNotify message;
        if (!message.ParseFromString(packet.payload)) {
          return false;
        }
        return callback(context, message);
      });
}

}  // namespace transform::battle
