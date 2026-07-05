package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wxdqing/go-transformgen/internal/define"
	"github.com/wxdqing/go-transformgen/internal/descriptor"
	"github.com/wxdqing/go-transformgen/internal/model"
	gotarget "github.com/wxdqing/go-transformgen/internal/target/go"
)

var errMissingArgument = errors.New("missing required argument")

type importMap map[string]string

func (m importMap) String() string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for key, value := range m {
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
}

func (m importMap) Set(value string) error {
	key, importPath, ok := strings.Cut(value, "=")
	if !ok || key == "" || importPath == "" {
		return fmt.Errorf("go-import must use key=value")
	}
	m[key] = importPath
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var protoSet string
	var definesDir string
	var target string
	var side string
	var outDir string
	var packageName string
	var templateDir string
	goImports := importMap{}

	fs := flag.NewFlagSet("transformgen", flag.ContinueOnError)
	fs.StringVar(&protoSet, "proto-set", "", "protoc descriptor set file")
	fs.StringVar(&definesDir, "defines-dir", "", "YAML definitions directory")
	fs.StringVar(&target, "target", "go", "target language")
	fs.StringVar(&side, "side", "requester,responder", "generated side")
	fs.StringVar(&outDir, "out", "", "output directory")
	fs.StringVar(&packageName, "package", "", "output package or namespace")
	fs.StringVar(&templateDir, "template-dir", "", "custom template directory")
	fs.Var(goImports, "go-import", "Go import override key=value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_ = templateDir
	if protoSet == "" || definesDir == "" || outDir == "" || packageName == "" {
		return errMissingArgument
	}
	if target != "go" {
		return fmt.Errorf("unsupported target %q", target)
	}

	desc, err := descriptor.Load(protoSet)
	if err != nil {
		return err
	}
	modules, err := define.LoadDir(definesDir)
	if err != nil {
		return err
	}
	built, err := model.Build(desc, modules)
	if err != nil {
		return err
	}
	files, err := gotarget.Render(built, gotarget.Options{
		Package: packageName,
		Sides:   splitCSV(side),
	})
	if err != nil {
		return err
	}
	for _, file := range files {
		path := filepath.Join(outDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, file.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
