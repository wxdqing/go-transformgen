#include "chat_responder.hpp"

namespace transform::chat {

Responder::Responder(PacketEndpoint& endpoint, ResponderHandler& handler)
    : endpoint_(&endpoint), handler_(&handler) {}

bool Responder::RegisterHandlers() {
  // Register one decode-invoke-reply callback per request id.
  if (!endpoint_->RegisterPacketCallback(
          ToMessageId(EMsgToServerType::SendChatRequest),
          [endpoint = endpoint_, handler = handler_](
              const PacketContext& context, const Packet& packet) {
            ::transform::example::SendChatRequest request;
            if (!request.ParseFromString(packet.payload)) {
              return false;
            }
            ::transform::example::SendChatResponse response;
            if (!handler->SendChat(context, request, response)) {
              return false;
            }
            Packet reply;
            reply.message_id = ToMessageId(EMsgToClientType::SendChatResponse);
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

bool Responder::SendChatMessage(const PacketContext& context,
                                  const ::transform::example::ChatMessageNotify& message) {
  // Notifies are uncorrelated; the endpoint allocates their downlink sequence.
  Packet packet;
  packet.message_id = ToMessageId(EMsgToClientType::ChatMessageNotify);
  if (!message.SerializeToString(&packet.payload)) {
    return false;
  }
  PacketContext notify_context = context;
  notify_context.request_id = 0;
  notify_context.packet_seq = 0;
  return endpoint_->SendPacket(notify_context, packet);
}

}  // namespace transform::chat
