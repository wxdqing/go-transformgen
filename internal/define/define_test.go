package define

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirLoadsModuleNameFromDefinition(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`version: 1
model_name: player_session
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
`)
	if err := os.WriteFile(filepath.Join(dir, "player_session.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	modules, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("len(modules) = %d, want 1", len(modules))
	}
	module := modules[0]
	if module.Name != "player_session" {
		t.Fatalf("module name = %q, want player_session", module.Name)
	}
	if module.CtxImportPath != "context" {
		t.Fatalf("module ctx import = %q, want context", module.CtxImportPath)
	}
	if module.RPCs[0].Method != "Heartbeat" || module.RPCs[0].Request != "transform.HeartbeatRequest" {
		t.Fatalf("rpc = %+v", module.RPCs[0])
	}
	if module.RPCs[0].CtxImportPath != "context" {
		t.Fatalf("rpc ctx import = %q, want context", module.RPCs[0].CtxImportPath)
	}
	if module.Notifies[0].Method != "BattleFinished" || module.Notifies[0].Message != "transform.BattleFinishedNotify" {
		t.Fatalf("notify = %+v", module.Notifies[0])
	}
	if module.Notifies[0].CtxImportPath != "context" {
		t.Fatalf("notify ctx import = %q, want context", module.Notifies[0].CtxImportPath)
	}
}

func TestLoadDirRejectsInvalidVersionAndFilename(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "player.yaml"), []byte("version: 2\nmodel_name: player\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDir(dir); !errors.Is(err, ErrUnsupportedVersion) {
			t.Fatalf("LoadDir() error = %v, want ErrUnsupportedVersion", err)
		}
	})

	t.Run("filename", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "Player.yaml"), []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDir(dir); !errors.Is(err, ErrInvalidModuleName) {
			t.Fatalf("LoadDir() error = %v, want ErrInvalidModuleName", err)
		}
	})
}

func TestLoadDirRejectsMissingOrMismatchedModelName(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "player.yaml"), []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDir(dir); !errors.Is(err, ErrInvalidModuleName) {
			t.Fatalf("LoadDir() error = %v, want ErrInvalidModuleName", err)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "player.yaml"), []byte("version: 1\nmodel_name: battle\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadDir(dir); !errors.Is(err, ErrInvalidModuleName) {
			t.Fatalf("LoadDir() error = %v, want ErrInvalidModuleName", err)
		}
	})
}
