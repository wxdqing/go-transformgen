using ProtoBuf;
using System.Runtime.CompilerServices;

public interface IRetErrorType
{ 
    public EMsgErrorType Ret { get; }
}

public interface IProtoBufToServer : IProtoBuf<EMsgToServerType> { }
public interface IProtoBufToClient : IProtoBuf<EMsgToClientType> { }

public partial class BattleFinishedNotify : IProtoBufToClient
{
    public const EMsgToClientType MsgType = EMsgToClientType.BattleFinishedNotify;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToClientType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class BattleStateNotify : IProtoBufToClient
{
    public const EMsgToClientType MsgType = EMsgToClientType.BattleStateNotify;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToClientType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class ChatMessageNotify : IProtoBufToClient
{
    public const EMsgToClientType MsgType = EMsgToClientType.ChatMessageNotify;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToClientType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class HeartbeatRequest : IProtoBufToServer
{
    public const EMsgToServerType MsgType = EMsgToServerType.HeartbeatRequest;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToServerType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class HeartbeatResponse : IProtoBufToClient, IRetErrorType
{
    public const EMsgToClientType MsgType = EMsgToClientType.HeartbeatResponse;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToClientType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class SendChatRequest : IProtoBufToServer
{
    public const EMsgToServerType MsgType = EMsgToServerType.SendChatRequest;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToServerType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class SendChatResponse : IProtoBufToClient, IRetErrorType
{
    public const EMsgToClientType MsgType = EMsgToClientType.SendChatResponse;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToClientType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class StartBattleRequest : IProtoBufToServer
{
    public const EMsgToServerType MsgType = EMsgToServerType.StartBattleRequest;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToServerType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}

public partial class StartBattleResponse : IProtoBufToClient, IRetErrorType
{
    public const EMsgToClientType MsgType = EMsgToClientType.StartBattleResponse;
    public const int MsgTypeInt = (int)MsgType;

    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public EMsgToClientType GetMsgType() => MsgType;
    [MethodImpl(MethodImplOptions.AggressiveInlining)]
    public int GetMsgTypeInt() => MsgTypeInt;
}
