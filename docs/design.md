# transformgen 设计文档

## 目标

`transformgen` 用于基于 proto message 与 YAML 方法定义生成 Go 协议代码。

设计目标：

- `transformgen` 作为独立仓库维护，Go module 为 `github.com/wxdqing/go-transformgen`。
- message 编号定义在 proto message 上，作为全局消息身份。
- YAML 只描述方法归属、模块归属、ctx 类型、request/response 关系。
- 生成模块接口，外部按模块名注册具体实现。
- 提供外部注册 request、response、notify message 与 handler 的能力。
- 支持 request、response、notify 三类消息。
- 同一套协议模型可用于 client/game，也可用于 game/battle 等服务间通信。

## 仓库与目录

`transformgen` 使用独立仓库：

```text
github.com/wxdqing/go-transformgen
```

本工程内可先以源码目录承载：

```text
tools/source/transformgen
```

并在 `go.work` 或调用方工程中通过 replace 指向本地目录。拆出独立仓库后，目录结构保持不变。

模块职责：

- 提供 `transformgen` CLI。
- 提供 descriptor/YAML 解析与中间模型。
- 提供多语言模板渲染框架。
- 通过 target 模板生成 runtime support，生成产物可自包含运行。
- 提供 proto option 定义，供业务 proto import。

项目生成代码默认只依赖业务 proto 生成代码和目标语言基础依赖，不依赖 `transformgen/runtime`。Go target 如需复用外部 runtime，可显式使用 `--runtime import` 并通过 `--go-import frame=...`、`--go-import registry=...` 指定外部包。

## Module 边界

独立 module 对外暴露两类稳定入口：

- `cmd/transformgen`：代码生成 CLI。
- `proto/options`：message option 定义。

`internal/*` 只服务 CLI 和模板渲染，不被生成代码 import。runtime 支持代码由 `internal/target/<lang>/templates` 产出到业务工程。

业务工程的典型依赖关系：

```text
resource/protocol/transform/*.proto
  -> import github.com/wxdqing/go-transformgen/proto/options/transform.proto

resource/protocol/src/transform/*.go
  -> contains generated frame/registry runtime support
```

如果本工程暂时使用本地源码，可以在 `go.work` 或业务 `go.mod` 中使用 replace：

```text
replace github.com/wxdqing/go-transformgen => ./tools/source/transformgen
```

## 协议分层

协议分为两层。

### Message 层

message 层由 proto 定义，负责：

- 消息结构。
- 消息编号。
- 消息类型：request、response、notify。

`HEAD.message_id` 必须等于当前 payload 对应 message 的编号。

### Method 层

method 层由 YAML 定义，负责：

- 模块名。
- 方法名。
- request 与 response 的对应关系。
- handler ctx 类型。
- notify handler 归属。

YAML 不再定义消息编号。

## Proto 源（一份）

业务只维护一份标准 `proto3` 文件。不在 proto 上声明 `message_id` / `message_kind` option。

业务 message 示例：

```proto
syntax = "proto3";

package transform.example;

option go_package = "github.com/wxdqing/go-transformgen/example/transform;examplepb";

message HeartbeatRequest {
  int64 client_time = 1;
  uint64 sequence = 2;
}

message HeartbeatResponse {
  int64 server_time = 1;
  int64 client_time = 2;
  uint64 sequence = 3;
}

message BattleFinishedNotify {
  uint64 battle_id = 1;
}
```

`request` / `response` / `notify` 角色由 YAML 定义；消息名需能自描述角色，支持：

- 后缀风格：`*Request` / `*Response` / `*Notify`
- TianLong 中缀风格：`MsgXxxReqYyy` / `MsgXxxResYyy` / `MsgXxxNtfYyy`

YAML 角色与命名推断冲突时生成失败。

## 消息 ID 生成（确定性哈希 + 锁定表）

运行时 message id 由 transformgen 分配，transformgen 是唯一权威来源；Go 与 C# 共用同一套最终 id。

首次（或未入表）消息按固定算法计算：

```text
id = |netHash(name) % 90000000| + band
band = 200000000  # request（client -> server）
band = 100000000  # response / notify（server -> client）
```

`netHash` 复刻 .NET Framework 的确定性 `String.GetHashCode`（双累加器 + `1566083941`，UTF-16 code unit，int32 溢出回绕）。方向由 YAML 角色决定：request 走 server 段，response/notify 走 client 段。与当前仍占用的 id 冲突时按升序探测 +1。

跨版本稳定性靠 `--msgid-lock` 指向的 YAML 锁定表（如 `msgid.lock.yaml`）：

- 表中已有短名 → 原样复用 id，不重算。
- 新消息 → 按上式分配，并避开表内仍占用的 id。
- 消息删除 → 从表中移除对应行，该 id 可被后续新消息复用。
- 每次生成后回写锁定表，使之与当前 message 全集同步。

锁定表应与协议源一并提交。

## Descriptor 输入

`transformgen` 不直接解析 `.proto` 源码，而是读取 `protoc` 产出的 descriptor set。

调用方生成 descriptor set 时必须包含 import 信息：

```text
protoc \
  --descriptor_set_out=transform.pbset \
  --include_imports \
  --include_source_info \
  -I <业务 proto 根目录> \
  -I <go-transformgen proto 根目录> \
  <业务 proto 文件列表>
```

descriptor set 中需要包含：

- 所有业务 message。
- `go_package`（以及可选的 `csharp_namespace`）。

生成器通过 descriptor set 读取 proto full name、Go/C# 类型信息，并由消息名推断 kind；运行时 id 在 model 构建时按哈希分配。

## YAML 定义

YAML 文件按模块拆分，放在：

```text
resource/protocol/transform/defines
```

文件名使用 `snake_case`：

```text
player.yaml
battle.yaml
chat_room.yaml
```

YAML 中必须显式写 `model_name`。文件名仍然使用 `snake_case`，且 basename 必须与 `model_name` 一致：

```text
player.yaml -> model_name: player
chat_room.yaml -> model_name: chat_room
```

这样定义文件在被移动、合并或外部工具读取时仍然自描述，同时用文件名一致性校验避免同一信息出现两份互相冲突的来源。

推荐格式：

```yaml
version: 1
model_name: player
ctx_import: context

rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: context.Context

notifies:
  - method: BattleFinished
    message: transform.BattleFinishedNotify
    ctx: context.Context
```

字段含义：

- `version`：定义文件格式版本，第一版固定为 `1`。
- `model_name`：模块名，必须是 snake_case，且必须与 YAML 文件名 basename 一致。
- `method`：生成的 Go 方法名，必须是合法导出标识符。
- `request`：proto request message 全名。
- `response`：proto response message 全名。
- `message`：notify message 全名。
- `ctx`：handler 第一个参数名固定为 `ctx`，该字段声明参数类型。
- `ctx_import`：文件级默认 ctx 类型 import path，放在 `model_name` 后，作用于该模块下所有 RPC/notify。单条 RPC/notify 仍可设置 `ctx_import` 覆盖默认值，用于少量特殊方法。`ctx: context.Context` 可以省略，生成器会自动补 `context`；自定义类型建议显式填写，例如 `ctx: grainactor.Context` 与 `ctx_import: apps/common/runtime/stateful/grainactor`。

## 生成内容

生成代码按目标语言和端类型输出。Go 目标建议输出到独立目录，不和 proto 生成文件混放：

```text
resource/protocol/src/transform/protocol
```

生成内容分为两类。

### 通用协议产物

通用协议产物与端类型无关：

- message_id 常量。
- message 元数据表。
- `RegisterMessages(reg)`。
- 可选的 request/response/notify 静态解析便捷函数。
- message 编码函数。

统一运行时分发以 `registry.MessageRegistry` 为准。静态解析函数只能作为生成包内的轻量 wrapper，不能维护第二套独立全局注册表。

Go target 默认生成 `protocol_messages.go`，包含全部 message_id 常量和 message 注册函数。

Go target 还会生成 `protocol.go` 作为统一入口：

- `NewModule(codec, params)`：构造协议模块并保存 fx group 收集到的业务模块。
- `Module.Start(ctx)`：创建 registry、注册 message，并注册所有业务模块 handler。
- `PackMessage(codec, head, msg)`：根据外部传入的 `frame.Head` 和 proto message，marshal body 后调用 `FrameCodec` 组成 packet。
- `Module.RegisterHandlers(modelName, impl)` / `RegisterHandlers(...)`：通过 `model_name` 注册具体模块实现。
- `Provider`：实现 `gitee.com/wxdqing/fx-bootstrap.Provider`，用于收集 fx group 中的模块实现、构造 `*Module`，并在 `OnStart` 中完成内部注册。
- `HandlerModule` / `HandlerModuleOut` / `HandlerModuleWithBean`：业务模块进入 fx group 的轻量包装。

`NewProtocol(nil)` 默认使用生成包内的 `PacketFrameCodec`。该 codec 由 target runtime 模板生成，生成协议入口不依赖 game peer/session；peer、gateway、battle RPC 等传输层只负责发送/接收 bytes。

### 端类型产物

端类型分为 request 端与 response 端。

request 端用于主动发起 RPC 或主动发送 notify：

- request 编码辅助函数。
- response 解析辅助函数。
- notify 编码辅助函数。
- 可选的 typed client stub。

response 端用于接收 request、调用实现、返回 response，或接收 notify：

- 模块接口。
- 模块注册函数。
- RPC dispatch 函数。
- notify dispatch 函数。
- response/notify 编码辅助函数。

Go target 按 `model_name` 拆分模块文件，生成 `<model_name>.go`，例如 `player.go`、`chat_room.go`。模块名常量、模块接口、handler 注册函数以及该模块相关的 request/response/notify helper 都放在对应模块文件中。

如果生成包和 proto message 包不同，Go target 根据 descriptor set 中的 `go_package` 自动 import proto 包，并用包名前缀引用 message 类型。

request 端和 response 端可以生成不同语言。例如：

- game 用 Go response 端处理 client 请求。
- battle 用 Go response 端处理 game 请求。
- client 用 TypeScript request 端发起请求并解析 response。
- battle 如果是其他语言，只需要生成对应语言的 request/response 端代码。

## 生成器组织

`transformgen` 内部按“解析 -> 中间模型 -> 模板渲染”组织。

推荐目录：

```text
tools/source/transformgen
  go.mod
  cmd/transformgen
  internal/config
  internal/descriptor
  internal/define
  internal/model
  internal/render
  internal/target/go
  internal/target/go/templates
  internal/target/csharp
  internal/target/csharp/templates
  docs
```

职责：

- `descriptor`：读取 proto descriptor，提取 message option、Go 包信息、message 全名。
- `define`：读取 YAML，校验模块、RPC、notify 定义。
- `model`：构建语言无关的中间模型。
- `render`：封装模板渲染。
- `target/<lang>`：负责不同语言的命名、import、文件布局。
- `target/<lang>/templates`：保存对应语言模板和 runtime support 模板。

第一阶段只实现 Go target 与 Go runtime；目录先按多语言留出边界，但不提前实现其他语言逻辑。

## 模板机制

代码生成使用 Go 标准库 `text/template`。

原则：

- 生成器逻辑只构建中间模型，不拼接大段代码字符串。
- 语言差异放在 target 与模板里。
- 模板只做展示，不做复杂业务判断。
- 每个输出文件对应一个模板，便于后续扩展其他语言。

Go target 默认模板示例：

```text
templates/go/constants.go.tmpl
templates/go/messages.go.tmpl
templates/go/requester.go.tmpl
templates/go/responder.go.tmpl
templates/go/frame_codec.go.tmpl
```

后续新增语言时，优先新增 target 和 templates，不修改核心解析模型。

## 生成器参数

第三方包、输出包名、模板目录等通过命令参数传入，避免写死在生成器里。

建议参数：

```text
--proto-set <path>            protoc descriptor set 文件
--defines-dir <path>          YAML 定义目录
--target go|csharp            目标语言
--side requester,responder    生成端类型
--runtime emit|import         runtime 处理方式，Go 默认 emit，C# 仅支持 emit
--out <dir>                   输出目录
--package <name>              输出包名或 namespace
--go-import frame=<import>    Go frame runtime import 映射
--go-import registry=<import> Go registry import 映射
--go-import proto=<import>    Go import 映射
--go-import context=<import>  Go import 映射
--go-import fx=<import>       Go fx import 映射
--go-import bootstrap=<import> Go fx-bootstrap import 映射
```

Go target 默认 import：

```text
proto=google.golang.org/protobuf/proto
context=context
fx=go.uber.org/fx
bootstrap=gitee.com/wxdqing/fx-bootstrap
```

`--runtime emit` 会把 Go runtime support 写入产出目录，生成代码不 import transformgen 自身的 runtime 包。`--runtime import` 只用于接入方已有外部 runtime 包的情况，必须显式提供 `frame` 与 `registry` import。

C# target 对齐 TianLong3 客户端约定，默认（`--runtime emit`）产出：

- 按 proto 源文件拆分的 protobuf-net message/enum 类（`[ProtoContract]` / `[ProtoMember]` / `IExtensible`，repeated 为 `List<T>`）
- `EMsgToServerType.cs`（request 方向枚举）
- `EMsgToClientType.cs`（response/notify 方向枚举）
- `EMsgType.cs`（`partial class` 绑定 `MsgType`/`MsgTypeInt`/`GetMsgType`，实现 `IProtoBufToServer`/`IProtoBufToClient`；含 `EMsgErrorType ret` 字段时附加 `IRetErrorType`）

字段覆盖：标量、bytes、enum、message 引用、repeated、map（`[ProtoMap] Dictionary<K,V>`）、oneof（protobuf-net `DiscriminatedUnion*` + `*OneofCase`）、proto2 group（`DataFormat.Group`）。proto3 `optional` 的合成 oneof 会被忽略并按普通字段输出。`--package`/`--side` 对 C# 无效（全局命名空间），且只支持 `emit`。

如果某个工程使用不同包路径，可以通过参数覆盖。

## 消息头包装

消息头包装不由协议生成代码写死，而是通过外部注入完成。

原因：

- client/game 与 game/battle 可能使用不同传输。
- request 端与 response 端可能是不同语言。
- `HEAD` 字段布局可能随链路变化。
- transformgen 只应关心 message_id 与 protobuf payload 的关系。

生成代码只依赖抽象的 frame codec。默认情况下该抽象由生成包内的 runtime support 提供。

Go runtime support 提供：

```go
type Head struct {
	MessageID uint32
	BodyLen   uint32
	RequestID uint64
	PacketSeq uint32
}

type FrameCodec interface {
	EncodeFrame(head Head, body []byte) ([]byte, func(), error)
	DecodeFrame(data []byte) (Head, []byte, func(), error)
}
```

说明：

- `BodyLen` 由 `EncodeFrame` 根据 `body` 计算或校验。
- 返回的 `func()` 用于释放池化 buffer；如果实现没有池化，返回空函数。
- 生成的 request/response 代码只构造 `Head` 和 protobuf body，然后调用注入的 `FrameCodec`。
- 具体网络发送、接收、粘包、压缩仍由接入层负责。

推荐生成两种使用方式。

request 端的低层 API 只构建 frame，不负责发送：

```go
func EncodeHeartbeatRequest(codec FrameCodec, requestID uint64, req *transformpb.HeartbeatRequest) ([]byte, func(), error)
func DecodeHeartbeatResponse(messageID uint32, payload []byte) (*transformpb.HeartbeatResponse, error)
```

request 端的可选 typed client 依赖外部注入的 round trip：

```go
type RequestIDAllocator interface {
	NextRequestID() uint64
}

type RoundTripper interface {
	RoundTrip(ctx context.Context, head Head, body []byte) (Head, []byte, func(), error)
}
```

typed client 只负责：

1. marshal request。
2. 通过 `RequestIDAllocator` 获取 request_id。
3. 填充 request message_id。
4. 调用 `RoundTripper`。
5. 校验返回的 response message_id。
6. unmarshal response。

response 端的 dispatch 不直接写网络，而是返回 response frame：

```go
func DispatchRequest(ctx any, codec FrameCodec, head Head, payload []byte) ([]byte, func(), error)
```

接入层收到返回值后决定如何发送。这样 response 端可以嵌入 gateway stream、actor session、battle RPC 连接等不同传输，不需要生成器了解具体网络。

Go runtime support 会提供一个默认实现：

```go
type PacketFrameCodec struct{}
```

编码规则由该实现决定，例如：

```text
message_id uint32
body_len   uint32
request_id uint64
packet_seq uint32
body       []byte
```

该默认实现只作为 Go 侧便利能力。其他链路可以注入自己的 `FrameCodec`，例如复用 `peer/codec/msgcodec` 的 HEAD 布局。

## 模块名常量

根据 YAML `model_name` 生成模块名常量。`model_name` 必须与 YAML 文件名 basename 一致：

```go
const ModelNamePlayer = "player"
const ModelNameChatRoom = "chat_room"
```

常量名由 `model_name` 转为 PascalCase。

## Message 常量

根据 proto message option 生成：

```go
const MessageIDHeartbeatRequest uint32 = 1001
const MessageIDHeartbeatResponse uint32 = 1002
const MessageIDBattleFinishedNotify uint32 = 3001
```

这些常量来自 message 本身，不来自 YAML。

## Runtime 注册表

生成包内的 runtime support 提供外部注册能力。

注册表分两类信息：

- message 注册：message_id、kind、构造函数、全名。
- handler 注册：request 或 notify 对应的业务处理函数。

### Message 注册

request、response、notify 都通过外部注册进入 registry。

Go runtime API 建议：

```go
type MessageKind uint8

const (
	MessageKindRequest MessageKind = 1
	MessageKindResponse MessageKind = 2
	MessageKindNotify MessageKind = 3
)

type MessageMeta struct {
	ID       uint32
	Kind     MessageKind
	FullName string
}

type MessageFactory func() proto.Message

type MessageRegistry interface {
	RegisterRequest(meta MessageMeta, newMessage MessageFactory) error
	RegisterResponse(meta MessageMeta, newMessage MessageFactory) error
	RegisterNotify(meta MessageMeta, newMessage MessageFactory) error

	ParseRequest(messageID uint32, payload []byte) (proto.Message, error)
	ParseResponse(messageID uint32, payload []byte) (proto.Message, error)
	ParseNotify(messageID uint32, payload []byte) (proto.Message, error)
	ParseMessage(messageID uint32, payload []byte) (proto.Message, MessageMeta, error)
}
```

生成代码不强制使用全局变量。推荐生成：

```go
func RegisterMessages(reg registry.MessageRegistry) error
```

示例：

```go
func RegisterMessages(reg registry.MessageRegistry) error {
	if err := reg.RegisterRequest(registry.MessageMeta{
		ID: MessageIDHeartbeatRequest,
		Kind: registry.MessageKindRequest,
		FullName: "transform.HeartbeatRequest",
	}, func() proto.Message { return &HeartbeatRequest{} }); err != nil {
		return err
	}
	return reg.RegisterResponse(registry.MessageMeta{
		ID: MessageIDHeartbeatResponse,
		Kind: registry.MessageKindResponse,
		FullName: "transform.HeartbeatResponse",
	}, func() proto.Message { return &HeartbeatResponse{} })
}
```

外部工程也可以注册非本生成包里的消息，只要 message_id 不冲突。

### Handler 注册

handler 注册面向 response 端，用于处理 request 和 notify。

Go runtime API 建议：

```go
type RequestHandler func(ctx any, req proto.Message) (proto.Message, error)
type NotifyHandler func(ctx any, msg proto.Message) error

type HandlerRegistry interface {
	RegisterRequestHandler(modelName string, requestID uint32, responseID uint32, handler RequestHandler) error
	RegisterNotifyHandler(modelName string, notifyID uint32, handler NotifyHandler) error

	DispatchRequest(ctx any, messageID uint32, payload []byte) (proto.Message, uint32, error)
	DispatchNotify(ctx any, messageID uint32, payload []byte) error
}
```

`DispatchRequest` 返回：

- response proto message。
- response message_id。
- error。

这样 response 端可以在接入层决定如何包装 HEAD 与发送 response。

实际实现建议提供一个同时包含 message 与 handler 能力的 registry：

```go
type Registry interface {
	MessageRegistry
	HandlerRegistry
}
```

`DispatchRequest` 和 `DispatchNotify` 内部需要先通过 message registry 解析 payload，再调用 handler registry 中的处理函数。因此默认实现应是单个 `registry.Registry`，而不是两张互不关联的表。

生成代码根据模块接口生成内部类型安全注册函数：

```go
func registerPlayerHandlers(reg registry.HandlerRegistry, impl Player) error
```

这些函数内部把同一个模块接口中的 request/notify 方法适配成 runtime 的 `RequestHandler` / `NotifyHandler`。它们是生成包内部实现细节，外部只通过 `HandlerModule` fx group 或 `Module.RegisterHandlers` 这个统一入口接入。

如果 YAML 中配置了自定义 ctx 类型，生成代码在适配函数内做类型断言：

```go
func registerPlayerHandlers(reg registry.HandlerRegistry, impl Player) error {
	return reg.RegisterRequestHandler(ModelNamePlayer, MessageIDHeartbeatRequest, MessageIDHeartbeatResponse,
		func(ctx any, req proto.Message) (proto.Message, error) {
			typedCtx, ok := ctx.(context.Context)
			if !ok {
				return nil, registry.ErrInvalidContextType
			}
			typedReq, ok := req.(*HeartbeatRequest)
			if !ok {
				return nil, registry.ErrInvalidMessageType
			}
			return impl.Heartbeat(typedCtx, typedReq)
		})
}
```

这样 runtime registry 不需要 import 业务 ctx 包，也能支持任意 Go ctx 类型。

### Registry 实例管理

runtime 可以提供默认 registry：

```go
func DefaultRegistry() *registry.Registry
```

但生成代码不应在 `init()` 中自动注册，避免隐藏全局状态。Go target 推荐由生成的 `Module.Start` 内部注册：

```go
func (m *Module) Start(ctx context.Context) error {
	reg := registry.New()
	_ = RegisterMessages(reg)
	// collect HandlerModule from fx group and register by ModuleName.
}
```

显式注册更利于测试、灰度、多协议版本和多语言边界。

## 模块接口

根据 YAML `model_name` 生成模块接口。

`player.yaml` 示例：

```go
type Player interface {
	Heartbeat(ctx context.Context, req *transformpb.HeartbeatRequest) (*transformpb.HeartbeatResponse, error)
	BattleFinished(ctx context.Context, msg *transformpb.BattleFinishedNotify) error
}
```

模块接口包含该模块下全部 request/response 和 notify 方法。这样外部通过 `model_name` 注册具体实现时，一个模块对应一个完整实现，初始化和 fx 装配都更直接。

## 注册方式

按模块生成内部 handler 注册函数：

```go
func registerPlayerHandlers(reg registry.HandlerRegistry, impl Player) error
```

同时生成统一入口：

```go
func RegisterHandlers(reg registry.HandlerRegistry, modelName string, impl any) error
```

统一入口内部根据 `modelName` 和接口类型校验实现是否正确。生成代码不持有全局实现，不在 `init()` 中注册。

Go target 还会生成 fx-bootstrap group 入口：

```go
type HandlerModule interface {
	ModuleName() string
	Module() any
}

type HandlerModuleOut struct {
	fx.Out
	Module HandlerModule `group:"transformgen_handler_modules"`
}

type HandlerModuleWithBean[T HandlerModule] struct {
	fx.Out
	Module HandlerModule `group:"transformgen_handler_modules"`
	Self T
}

func NewHandlerModuleWithBean[T HandlerModule](module T) HandlerModuleWithBean[T]

type Provider struct {
	bootstrap.NopHook
	Codec frame.FrameCodec
}

func (p Provider) Register() any
```

业务模块实现自身业务接口，同时实现 `HandlerModule`：

```go
func (m *PlayerModule) ModuleName() string { return protocolpb.ModelNamePlayer }
func (m *PlayerModule) Module() any       { return m }
```

业务模块 Provider 可以直接返回 `protocolpb.NewHandlerModuleWithBean(m)`，既进入协议 group，也保留 typed bean 给其他模块注入。生成的 `Provider` 构造 `*Module` 时会收集 group 中的所有 `HandlerModule`；在 `OnStart` 调用 `Start` 时，内部注册 message_id，并通过 `ModuleName()` 区分模块调用 `RegisterHandlers`。这样启动代码只需要把协议 Provider 和业务模块 Provider 放进 fx-bootstrap，不需要手动逐个注册。

## 解析函数

解析由 `registry.MessageRegistry` 提供，分为三类。

```go
func ParseRequest(messageID uint32, payload []byte) (proto.Message, error)
func ParseResponse(messageID uint32, payload []byte) (proto.Message, error)
func ParseNotify(messageID uint32, payload []byte) (proto.Message, error)
```

规则：

- request 只能解析 `MESSAGE_KIND_REQUEST`。
- response 只能解析 `MESSAGE_KIND_RESPONSE`。
- notify 只能解析 `MESSAGE_KIND_NOTIFY`。
- kind 不匹配时返回明确错误。

如果调用方只知道 message_id，不知道 kind，可以提供：

```go
func ParseMessage(messageID uint32, payload []byte) (proto.Message, MessageMeta, error)
```

生成包的静态解析函数如果存在，也应委托给调用方传入的 registry，或只在本包内部构建临时 registry。不能和 runtime registry 形成两套长期状态。

## RPC 流程

请求：

```text
HEAD.message_id = request message_id
HEAD.request_id != 0
payload = Request protobuf bytes
```

处理：

1. 根据 `HEAD.message_id` 查 message 元数据。
2. 校验 message kind 是 request。
3. 反序列化 request。
4. 根据 YAML 生成的路由找到模块实现与方法。
5. 调用：

```go
resp, err := impl.Heartbeat(ctx, req)
```

响应：

```text
HEAD.message_id = response message_id
HEAD.request_id = 原 request_id
HEAD.packet_seq = 原 packet_seq 或调用方指定值
payload = Response protobuf bytes
```

如果 handler 返回 error，第一版不生成通用 wire error response。dispatch 将 error 原样返回给接入层，由接入层决定关闭连接、记录日志或转换成业务错误响应。

## Notify 流程

主动通知：

```text
HEAD.message_id = notify message_id
HEAD.request_id = 0
payload = Notify protobuf bytes
```

notify 可用于：

- game 主动通知 client。
- battle 主动通知 game。
- 任意服务间无响应事件。

生成编码辅助函数：

```go
func EncodeBattleFinishedNotify(head Head, msg *transformpb.BattleFinishedNotify) ([]byte, error)
```

调用方负责通过已有网络会话发送编码后的 payload。

如果当前进程需要处理 notify，则通过 notify handler 分发：

```go
func DispatchNotify(ctx any, messageID uint32, payload []byte) error
```

## request_id 语义

`request_id` 表示请求关联关系，不表示消息类型。

规则：

- RPC request：`request_id != 0`。
- RPC response：`request_id = 原 request_id`。
- notify：`request_id = 0`。

message 类型由 `HEAD.message_id` 与 proto option 决定。

## ctx 类型

YAML 中的 `ctx` 直接决定 Go response 端生成接口签名。request 端不使用该字段。参数名固定为 `ctx`，类型由 `ctx` 字段决定；类型所需 import 优先使用单条 RPC/notify 的 `ctx_import`，否则使用文件级 `ctx_import`。

示例：

```yaml
version: 1
model_name: player
ctx_import: context
rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: context.Context
```

生成：

```go
Heartbeat(ctx context.Context, req *transformpb.HeartbeatRequest) (*transformpb.HeartbeatResponse, error)
```

如果配置更深的类型：

```yaml
version: 1
model_name: player
ctx_import: apps/common/runtime/stateful/grainactor
rpcs:
  - method: Heartbeat
    request: transform.HeartbeatRequest
    response: transform.HeartbeatResponse
    ctx: grainactor.Context
```

生成代码必须 import 对应包，并生成：

```go
Heartbeat(ctx grainactor.Context, req *transformpb.HeartbeatRequest) (*transformpb.HeartbeatResponse, error)
```

如果类型不存在，生成代码编译失败即可暴露配置错误。生成器也可以在后续增强静态校验。

非 Go target 不直接复用 Go 的 `ctx` 字符串。后续新增其他语言时，由对应 target 决定是否忽略、映射或要求额外配置。

## 与现有代码的关系

当前 game 侧 `apps/game/actor/player/router.go` 通过 payload 猜测消息类型。接入 `transformgen` 后，应改为：

- 使用 `HEAD.message_id` 显式分发。
- 不再通过反复 `proto.Unmarshal` 猜测 request 类型。
- heartbeat 变成普通 RPC 方法。

peer 底层的握手消息仍保留现有 message_id。业务协议 message_id 需要避开底层保留范围。

## 校验规则

生成器应至少校验：

- YAML `version` 是支持的版本。
- YAML 文件名是 snake_case。
- YAML `model_name` 是 snake_case，且与文件名 basename 一致。
- method 是合法导出 Go 标识符。
- request message 存在且 kind 是 request。
- response message 存在且 kind 是 response。
- notify message 存在且 kind 是 notify。
- message_id 全局唯一。
- 同一个 request 只能绑定一个 RPC 方法。
- YAML 中引用的 proto package/message 能解析到生成 Go 类型。
- 模块名常量无冲突。

## 错误边界

runtime registry 第一版需要提供清晰错误，便于接入层判断问题来源：

- `ErrDuplicateMessageID`：重复注册 message_id。
- `ErrUnknownMessageID`：解析或 dispatch 时找不到 message_id。
- `ErrMessageKindMismatch`：按 request/response/notify 解析时 kind 不匹配。
- `ErrDuplicateHandler`：同一个 request 或 notify 重复注册 handler。
- `ErrHandlerNotFound`：message 存在，但没有对应 handler。
- `ErrInvalidContextType`：生成适配器中的 ctx 类型断言失败。
- `ErrInvalidMessageType`：生成适配器中的 proto message 类型断言失败。

错误应可通过 `errors.Is` 判断。生成代码不吞掉这些错误，直接返回给接入层。

## 实施顺序

建议按以下顺序实现，避免一次性铺太大：

1. 建立独立 module 骨架：`github.com/wxdqing/go-transformgen`。
2. 添加 proto option 定义与生成方式。
3. 实现 Go runtime support 模板：message 注册、解析、handler 注册、dispatch。
4. 实现 Go frame support 模板：`Head`、`FrameCodec`、`PacketFrameCodec`。
5. 实现 descriptor set 读取，提取 message_id、kind、Go 类型信息。
6. 实现 YAML 读取与校验。
7. 构建语言无关中间模型。
8. 实现 Go templates：constants、messages、responder、requester。
9. 在 `resource/protocol/transform` 中用 heartbeat 做最小闭环。
10. 替换 `apps/game/actor/player/router.go` 中按 payload 猜类型的逻辑。

第一条业务链路只要求 heartbeat request/response 跑通；notify 可用一个简单测试消息验证编码和 dispatch，不急着接入真实业务。

## 测试策略

`go-transformgen` 自身测试：

- registry 注册 request/response/notify 成功。
- 重复 message_id 返回 `ErrDuplicateMessageID`。
- kind 不匹配返回 `ErrMessageKindMismatch`。
- unknown message_id 返回 `ErrUnknownMessageID`。
- handler dispatch 能正确解析 request 并返回 response_id。
- ctx 类型错误返回 `ErrInvalidContextType`。
- `PacketFrameCodec` encode/decode round trip，并验证 release 生命周期。
- descriptor reader 能读取 option message_id、message_kind、go_package。
- YAML parser 校验 `version`、文件名、method、message 引用。

生成器 golden tests：

- 给定固定 proto descriptor 和 YAML，生成 Go 文件与 golden 文件一致。
- `requester` only、`responder` only、`requester,responder` 三种 side 输出稳定。
- 自定义 `--go-import` 能正确影响 import。

集成测试：

- heartbeat request 使用 request 端编码。
- response 端 dispatch 到 `Player.Heartbeat`。
- response 使用 response message_id 回包。
- request 端按 response message_id 解析响应。
- notify 使用 message_id + `request_id=0` 编码，并能被 notify handler 接收。

## 迁移策略

现有 transform heartbeat 迁移建议：

1. 先给 `HeartbeatRequest`、`HeartbeatResponse` 增加 message option。
2. 新增 `resource/protocol/transform/defines/player.yaml`。
3. 生成 transform Go 代码，但暂不删除旧 router 分支。
4. 在测试中用新 generated dispatch 跑通 heartbeat。
5. 将 game router 改为按 `HEAD.message_id` dispatch。
6. 删除旧的“尝试反序列化 heartbeat，再尝试 ping”的猜测逻辑。

迁移期间不改变 gateway/stream 的会话生命周期，只替换 app payload 的协议分发方式。

## 第一阶段范围

第一阶段只实现必要能力：

- 独立 Go module：`github.com/wxdqing/go-transformgen`。
- Go runtime registry 与 frame codec 抽象。
- proto message option。
- YAML 读取。
- 生成模块接口与注册函数。
- 生成 message_id 常量与元数据。
- 生成 request/response/notify parse 函数。
- 生成 RPC dispatch。
- 生成 notify encode/dispatch 基础能力。

暂不实现：

- 跨语言客户端代码生成。
- 复杂错误码框架。
- 动态热加载协议。
- 多版本协议兼容策略。

这些能力等真实需求出现后再补。
