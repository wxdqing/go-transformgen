# transformgen Implementation Guide

## TDD/BDD Policy

All production behavior is implemented test-first.

For each behavior:

1. Write a focused failing test.
2. Run the exact test and confirm it fails for the expected reason.
3. Implement the smallest code needed to pass.
4. Run the focused test and then `go test ./...`.
5. Refactor only after green.

BDD-style tests describe behavior from the caller perspective:

- registry callers register and parse messages.
- response-side callers dispatch request payloads into typed handlers.
- request-side callers encode frames without knowing transport details.
- generator callers pass descriptor/YAML input and receive deterministic files.

## Generated Runtime Support APIs

### Registry Support

Generated Go runtime support provides:

- `MessageKind`
- `MessageMeta`
- `MessageFactory`
- `RequestHandler`
- `NotifyHandler`
- `MessageRegistry`
- `HandlerRegistry`
- `Registry`
- `NewRegistry()`

Required errors:

- `ErrDuplicateMessageID`
- `ErrUnknownMessageID`
- `ErrMessageKindMismatch`
- `ErrDuplicateHandler`
- `ErrHandlerNotFound`
- `ErrInvalidContextType`
- `ErrInvalidMessageType`

### Frame Support

Generated Go runtime support provides:

- `Head`
- `FrameCodec`
- `PacketFrameCodec`

`PacketFrameCodec` wire layout:

```text
message_id uint32
body_len   uint32
request_id uint64
packet_seq uint32
body       []byte
```

The returned release function owns pooled buffers.

## Internal Generator APIs

### `internal/define`

Reads one or more YAML files.

```go
func LoadDir(dir string) ([]Module, error)
```

Module names come from required YAML `model_name` values. File basenames must be snake_case and must match `model_name`. YAML `version` must be `1`. Module files may set file-level `ctx_import` after `model_name` to tell the Go target which package to import for all `ctx` parameter types in that module. RPC and notify definitions may still set `ctx_import` to override the module default.

### `internal/descriptor`

Reads descriptor set files.

```go
func Load(path string) (*Set, error)
```

The set exposes protobuf full names, message IDs, message kinds, Go import paths, and Go type names. When a proto file sets file-level `message_id_min` and `message_id_max` options, descriptor loading rejects messages whose `message_id` is outside that closed range. The Go target uses descriptor `go_package` data to import proto message packages even when generated protocol code lives in a different package.

### `internal/model`

Joins descriptor messages with YAML modules. This package is the current IR layer: it owns the language-neutral model consumed by targets.

```go
func Build(desc *descriptor.Set, modules []define.Module) (*Model, error)
```

It validates message existence, kind matching, duplicate IDs, and duplicate request bindings.

### `internal/render`

Small wrapper around `text/template`.

```go
func Render(name, text string, data any) ([]byte, error)
```

### `internal/target/go`

Renders deterministic Go files from `model.Model`.

```go
func Render(m *model.Model, opts Options) ([]File, error)
```

The Go target uses `text/template` with template files:

```text
internal/target/go/templates/messages.go.tmpl
internal/target/go/templates/module.go.tmpl
internal/target/go/templates/protocol.go.tmpl
```

The protocol template emits `protocol.go` with `HandlerModule`, `HandlerModuleOut`, `HandlerModuleWithBean`, `Provider`, `Module`, `NewModule`, `NewProtocol`, `PackMessage`, `Module.RegisterHandlers`, and the package-level `RegisterHandlers` switch by model name. `Provider` implements `gitee.com/wxdqing/fx-bootstrap.Provider`, collects `HandlerModule` values from the `transformgen_handler_modules` fx group in the constructor, and registers messages plus handlers in `OnStart` through `Module.Start`. `NewProtocol(nil)` remains as a convenience wrapper and uses the generated `PacketFrameCodec`.

The messages template emits `protocol_messages.go` with message constants and `RegisterMessages`.

The module template emits one `<model_name>.go` file per module with the model constant, a single module interface containing all request and notify methods, internal responder handler registration, requester request encoding, response decoding, and notify encoding. Module imports include proto message packages from descriptors and ctx packages from `ctx_import`; `context.Context` is auto-imported as `context` for compatibility.

When the generated package differs from the proto message package, the Go target imports the proto package from descriptor `go_package` and qualifies message types with that package name.

## CLI Behavior

`cmd/transformgen` accepts:

```text
--proto-set <path>
--defines-dir <path>
--target go|csharp
--side requester,responder
--runtime import|emit
--out <dir>
--package <name>
--go-import key=value
```

Go defaults to `--runtime emit`, which writes runtime support files into the output package. `--runtime import` is only for projects that provide their own external frame/registry packages through `--go-import frame=...` and `--go-import registry=...`. C# supports runtime emit and can generate requester/responder protocol helpers without depending on Go runtime packages.

## BDD Acceptance Scenarios

1. Given registered request and response messages, when request bytes are dispatched, then the matching handler is invoked and returns the response message ID.
2. Given a notify message and handler, when notify bytes are dispatched, then the notify handler receives the typed message and no response is returned.
3. Given a wrong kind parse call, when parsing a response as a request, then `ErrMessageKindMismatch` is returned.
4. Given a YAML file with version `1`, when loading definitions, then the module name is derived from the filename.
5. Given descriptor messages and YAML RPC definitions, when building the model, then RPC request and response types are linked.
6. Given the model, when rendering Go target files, then generated files are deterministic and contain registration functions.

## Implementation Notes

- Prefer explicit maps over reflection-heavy registries.
- Runtime registry may use mutexes for safe startup and test use.
- Generated handler adapters perform ctx and proto message type assertions.
- Descriptor code may use `protodesc.NewFiles` and `protoregistry.Files`.
- YAML parser uses `gopkg.in/yaml.v3`; this dependency has clear value and is isolated to generator internals.
