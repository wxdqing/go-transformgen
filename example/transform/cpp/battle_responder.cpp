#include "battle_responder.hpp"

namespace transform::battle {

Responder::Responder(PacketEndpoint& endpoint, ResponderHandler& handler)
    : endpoint_(&endpoint), handler_(&handler) {}

bool Responder::RegisterHandlers() {
  // Register one decode-invoke-reply callback per request id.
  if (!endpoint_->RegisterPacketCallback(
          ToMessageId(EMsgToServerType::StartBattleRequest),
          [endpoint = endpoint_, handler = handler_](
              const PacketContext& context, const Packet& packet) {
            ::transform::example::StartBattleRequest request;
            if (!request.ParseFromString(packet.payload)) {
              return false;
            }
            ::transform::example::StartBattleResponse response;
            if (!handler->StartBattle(context, request, response)) {
              return false;
            }
            Packet reply;
            reply.message_id = ToMessageId(EMsgToClientType::StartBattleResponse);
            if (!response.SerializeToString(&reply.payload)) {
              return false;
            }
            // Preserve request correlation but let the endpoint allocate the
            // server-to-client sequence independently from the uplink packet.
            PacketContext reply_context = context;
            reply_context.packet_seq = 0;
            return endpoint->SendPacket(reply_context, reply);
          })) {
    return false;
  }
  return true;
}

bool Responder::SendBattleState(const PacketContext& context,
                                  const ::transform::example::BattleStateNotify& message) {
  // Notifies are uncorrelated; the endpoint allocates their downlink sequence.
  Packet packet;
  packet.message_id = ToMessageId(EMsgToClientType::BattleStateNotify);
  if (!message.SerializeToString(&packet.payload)) {
    return false;
  }
  PacketContext notify_context = context;
  notify_context.request_id = 0;
  notify_context.packet_seq = 0;
  return endpoint_->SendPacket(notify_context, packet);
}

}  // namespace transform::battle
