#include "chat_requester.hpp"

#include <utility>

namespace transform::chat {

Requester::Requester(PacketEndpoint& endpoint) : endpoint_(&endpoint) {}

bool Requester::SendSendChat(const PacketContext& context,
                                  const ::transform::example::SendChatRequest& request) {
  // Serialize the payload and forward it with the request message id.
  Packet packet;
  packet.message_id = ToMessageId(EMsgToServerType::SendChatRequest);
  if (!request.SerializeToString(&packet.payload)) {
    return false;
  }
  return endpoint_->SendPacket(context, packet);
}

bool Requester::OnSendChatResponse(SendChatResponseCallback callback) {
  if (callback == nullptr) {
    return false;
  }
  // Decode the payload before invoking the typed callback.
  return endpoint_->RegisterPacketCallback(
      ToMessageId(EMsgToClientType::SendChatResponse),
      [callback = std::move(callback)](const PacketContext& context,
                                       const Packet& packet) {
        ::transform::example::SendChatResponse response;
        if (!response.ParseFromString(packet.payload)) {
          return false;
        }
        return callback(context, response);
      });
}

bool Requester::OnChatMessage(ChatMessageCallback callback) {
  if (callback == nullptr) {
    return false;
  }
  // Decode the payload before invoking the typed callback.
  return endpoint_->RegisterPacketCallback(
      ToMessageId(EMsgToClientType::ChatMessageNotify),
      [callback = std::move(callback)](const PacketContext& context,
                                       const Packet& packet) {
        ::transform::example::ChatMessageNotify message;
        if (!message.ParseFromString(packet.payload)) {
          return false;
        }
        return callback(context, message);
      });
}

}  // namespace transform::chat
