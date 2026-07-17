package msgidlock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingReturnsEmpty(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestSaveLoadRoundTripSorted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "msgid.lock.yaml")
	if err := Save(path, map[string]uint32{
		"ZebraRequest":  3,
		"AlphaResponse": 1,
		"MiddleNotify":  2,
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	alpha := strings.Index(text, "AlphaResponse:")
	middle := strings.Index(text, "MiddleNotify:")
	zebra := strings.Index(text, "ZebraRequest:")
	if !(alpha >= 0 && middle > alpha && zebra > middle) {
		t.Fatalf("messages not sorted by name:\n%s", text)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["AlphaResponse"] != 1 || got["MiddleNotify"] != 2 || got["ZebraRequest"] != 3 {
		t.Fatalf("Load() = %#v", got)
	}
}
