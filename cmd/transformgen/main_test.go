package main

import (
	"errors"
	"testing"
)

func TestRunRequiresCoreArguments(t *testing.T) {
	err := run([]string{"--target", "go"})
	if !errors.Is(err, errMissingArgument) {
		t.Fatalf("run() error = %v, want errMissingArgument", err)
	}
}

func TestParseGoImportFlag(t *testing.T) {
	imports := importMap{}
	if err := imports.Set("registry=example.com/registry"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := imports["registry"]; got != "example.com/registry" {
		t.Fatalf("registry import = %q", got)
	}
}
