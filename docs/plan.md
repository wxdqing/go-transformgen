# transformgen Development Plan

## Goal

Build the first-stage `github.com/wxdqing/go-transformgen` module from `design.md`: generated runtime support templates, proto options, YAML definition parsing, descriptor loading, a language-neutral model, and Go/C# code generators.

## Scope

This plan implements the foundation only. Runtime support is emitted into generated protocol packages by default.

## Deliverables

- Independent Go module under `tools/source/transformgen`.
- `proto/options/transform.proto` plus generated Go option code.
- Go runtime support templates with external request/response/notify registration and dispatch.
- Go frame support templates with injectable frame codec and a standard-library default codec.
- YAML parser for module definition files, with explicit `model_name` and filename consistency validation.
- Descriptor reader for protobuf descriptor sets.
- Minimal model builder joining descriptor messages and YAML methods.
- Minimal Go target renderer using `text/template` and real `.tmpl` files, emitting common message files plus per-module files into a generated protocol package.
- CLI entrypoint `cmd/transformgen`.
- Tests for registry, frame codec, YAML parsing, descriptor loading, model building, and rendering.

## Architecture

The module is split into proto option packages and internal generator packages.

- Runtime support is emitted into generated code by target templates.
- Internal packages are used only by `cmd/transformgen`.
- The generator reads descriptor sets and YAML, builds an intermediate model, and renders target-specific templates.

## Package Layout

```text
tools/source/transformgen
  go.mod
  cmd/transformgen
  proto/options
  internal/define
  internal/descriptor
  internal/model
  internal/render
  internal/target/go
  internal/target/go/templates
  internal/target/csharp
  internal/target/csharp/templates
  docs
```

## Implementation Phases

1. Generated runtime registry and frame codec templates.
2. Proto options and descriptor loading.
3. YAML definitions and model building.
4. Go rendering and CLI.
5. Verification and docs alignment.

The IR layer is `internal/model`. It is intentionally named after its responsibility in Go code, but it serves as the intermediate representation between descriptor/YAML parsing and language targets.

## Constraints

- Keep abstractions small and concrete.
- Do not auto-register generated code in `init()`.
- Generated code depends on proto options at schema-generation time and emits runtime support by default.
- Header wrapping is injected through the generated `FrameCodec`.
- Use `text/template` for generated files.
- YAML format version is fixed to `1` for the first implementation.

## Verification Commands

```bash
cd tools/source/transformgen && go test ./...
```

When integrated into the workspace:

```bash
go test ./tools/source/transformgen/...
```

The module can be tested independently first; workspace integration comes after the local module tests are stable.
