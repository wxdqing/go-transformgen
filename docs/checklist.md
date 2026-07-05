# transformgen Implementation Checklist

## Documentation

- [x] Create development plan from `design.md`.
- [x] Create implementation guide with TDD/BDD details.
- [x] Create this checklist.

## Module Setup

- [x] Create `tools/source/transformgen/go.mod`.
- [x] Add public package skeletons for `runtime/registry` and `runtime/frame`.
- [x] Add internal package skeletons for `define`, `descriptor`, `model`, `render`, and `target/go`.
- [x] Add `cmd/transformgen` entrypoint.

## Runtime Registry

- [x] Write failing tests for message registration and parsing.
- [x] Implement message registration and parsing.
- [x] Write failing tests for duplicate IDs and kind mismatch.
- [x] Implement registry errors.
- [x] Write failing tests for request and notify handler dispatch.
- [x] Implement handler registration and dispatch.

## Runtime Frame

- [x] Write failing tests for `PacketFrameCodec` encode/decode round trip.
- [x] Implement `Head`, `FrameCodec`, and `PacketFrameCodec`.
- [x] Write failing tests for malformed frame body length.
- [x] Implement malformed frame validation.

## Proto Options

- [x] Add `proto/options/transform.proto`.
- [x] Add generated or handwritten Go option bindings.
- [x] Write descriptor test fixture using message options.
- [x] Add file-level message ID range options and descriptor validation.

## YAML Definitions

- [x] Write failing tests for loading a valid YAML module.
- [x] Implement YAML loader.
- [x] Write failing tests for invalid version and invalid filename.
- [x] Implement YAML validation.
- [x] Write failing tests for missing or mismatched `model_name`.
- [x] Require `model_name` and validate it matches the YAML filename.

## Descriptor Loading

- [x] Write failing tests for descriptor loading from a fixture.
- [x] Implement descriptor set loader.
- [x] Extract message ID, kind, full name, Go import path, and Go type name.

## Model Building

- [x] Write failing tests for linking YAML RPCs to descriptor messages.
- [x] Implement model builder.
- [x] Write failing tests for missing messages and kind mismatches.
- [x] Implement model validation.

## Rendering and CLI

- [x] Write failing tests for template rendering.
- [x] Implement `internal/render`.
- [x] Write failing golden test for Go target rendering.
- [x] Implement minimal Go target renderer.
- [x] Move Go target rendering to a real `text/template` file.
- [x] Generate request encoding, response decoding, notify encoding, and notify handler registration.
- [x] Split Go output into `protocol_messages.go` and one `<model_name>.go` per module.
- [x] Support generated Go packages that live in a separate directory from proto message packages.
- [x] Generate `protocol.go` as the package-level entry with default go-utils-backed packet codec injection.
- [x] Generate one module interface containing both RPC and notify methods.
- [x] Generate an fx-bootstrap `Provider` for `*Protocol`.
- [x] Collect module implementations through a generated fx group `HandlerModule`.
- [x] Register messages and handlers inside generated `Module.Start`.
- [x] Generate `NewHandlerModuleWithBean` for concise business module providers.
- [x] Move realistic fx group business implementations into `example/demo`.
- [x] Keep per-module handler registration helpers internal to generated code.
- [x] Support ctx type imports through file-level YAML `ctx_import`, with per-entry override compatibility.
- [x] Import proto message packages from descriptor `go_package` when generated protocol code is in another package.
- [x] Write failing CLI argument test or smoke test.
- [x] Implement CLI.

## Verification

- [x] Run `cd tools/source/transformgen && go test ./...`.
- [x] Ensure no generated implementation relies on hidden `init()` registration.
- [x] Update docs if implementation diverges from design.
