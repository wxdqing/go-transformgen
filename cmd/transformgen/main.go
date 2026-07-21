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
	"github.com/wxdqing/go-transformgen/internal/msgidlock"
	cpptarget "github.com/wxdqing/go-transformgen/internal/target/cpp"
	csharptarget "github.com/wxdqing/go-transformgen/internal/target/csharp"
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
	var msgidLock string
	var target string
	var side string
	var outDir string
	var packageName string
	var runtimeMode string
	var cppProtoIncludePrefix string
	goImports := importMap{}

	fs := flag.NewFlagSet("transformgen", flag.ContinueOnError)
	fs.StringVar(&protoSet, "proto-set", "", "protoc descriptor set file")
	fs.StringVar(&definesDir, "defines-dir", "", "YAML definitions directory")
	fs.StringVar(&msgidLock, "msgid-lock", "", "message id lock file (yaml); created/updated to keep ids stable")
	fs.StringVar(&target, "target", "go", "target language")
	fs.StringVar(&side, "side", "requester,responder", "generated side")
	fs.StringVar(&outDir, "out", "", "output directory")
	fs.StringVar(&packageName, "package", "", "output package or namespace")
	fs.StringVar(&runtimeMode, "runtime", "emit", "runtime mode: emit or import")
	fs.StringVar(&cppProtoIncludePrefix, "cpp-proto-include-prefix", "", "C++ include prefix for protoc *.pb.h headers")
	fs.Var(goImports, "go-import", "Go import override key=value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if protoSet == "" || definesDir == "" || outDir == "" || packageName == "" {
		return errMissingArgument
	}
	desc, err := descriptor.Load(protoSet)
	if err != nil {
		return err
	}
	modules, err := define.LoadDir(definesDir)
	if err != nil {
		return err
	}
	var locked map[string]uint32
	if msgidLock != "" {
		locked, err = msgidlock.Load(msgidLock)
		if err != nil {
			return err
		}
	}
	built, nextLock, err := model.Build(desc, modules, locked)
	if err != nil {
		return err
	}
	files, err := renderTarget(target, built, desc, packageName, splitCSV(side), runtimeMode, goImports, cppProtoIncludePrefix)
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
	if msgidLock != "" {
		if err := os.MkdirAll(filepath.Dir(msgidLock), 0o755); err != nil {
			return err
		}
		if err := msgidlock.Save(msgidLock, nextLock); err != nil {
			return err
		}
	}
	return nil
}

type outputFile struct {
	Path    string
	Content []byte
}

func renderTarget(target string, built *model.Model, desc *descriptor.Set, packageName string, sides []string, runtimeMode string, goImports importMap, cppProtoIncludePrefix string) ([]outputFile, error) {
	switch target {
	case "go":
		files, err := gotarget.Render(built, gotarget.Options{
			Package: packageName,
			Sides:   sides,
			Imports: goImportPaths(goImports),
			Runtime: gotarget.RuntimeMode(runtimeMode),
		})
		if err != nil {
			return nil, err
		}
		return goOutputFiles(files), nil
	case "csharp":
		files, err := csharptarget.Render(built, desc, csharptarget.Options{
			Namespace: packageName,
			Sides:     sides,
			Runtime:   csharptarget.RuntimeMode(runtimeMode),
		})
		if err != nil {
			return nil, err
		}
		return csharpOutputFiles(files), nil
	case "cpp":
		files, err := cpptarget.Render(built, cpptarget.Options{
			Namespace:          packageName,
			Sides:              sides,
			Runtime:            cpptarget.RuntimeMode(runtimeMode),
			ProtoIncludePrefix: cppProtoIncludePrefix,
		})
		if err != nil {
			return nil, err
		}
		return cppOutputFiles(files), nil
	default:
		return nil, fmt.Errorf("unsupported target %q", target)
	}
}

// cppOutputFiles converts C++ target files to CLI output entries.
func cppOutputFiles(files []cpptarget.File) []outputFile {
	out := make([]outputFile, 0, len(files))
	for _, file := range files {
		out = append(out, outputFile{Path: file.Path, Content: file.Content})
	}
	return out
}

func goOutputFiles(files []gotarget.File) []outputFile {
	out := make([]outputFile, 0, len(files))
	for _, file := range files {
		out = append(out, outputFile{Path: file.Path, Content: file.Content})
	}
	return out
}

func csharpOutputFiles(files []csharptarget.File) []outputFile {
	out := make([]outputFile, 0, len(files))
	for _, file := range files {
		out = append(out, outputFile{Path: file.Path, Content: file.Content})
	}
	return out
}

func goImportPaths(values importMap) gotarget.ImportPaths {
	return gotarget.ImportPaths{
		Frame:     values["frame"],
		Registry:  values["registry"],
		Proto:     values["proto"],
		Context:   values["context"],
		FX:        values["fx"],
		Bootstrap: values["bootstrap"],
	}
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
