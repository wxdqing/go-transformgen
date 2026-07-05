package define

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const Version = 1

var moduleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

type Module struct {
	Name          string
	Version       int
	CtxImportPath string
	RPCs          []RPC
	Notifies      []Notify
}

type RPC struct {
	Method        string `yaml:"method"`
	Request       string `yaml:"request"`
	Response      string `yaml:"response"`
	Ctx           string `yaml:"ctx"`
	CtxImportPath string `yaml:"ctx_import"`
}

type Notify struct {
	Method        string `yaml:"method"`
	Message       string `yaml:"message"`
	Ctx           string `yaml:"ctx"`
	CtxImportPath string `yaml:"ctx_import"`
}

type fileDefinition struct {
	Version       int      `yaml:"version"`
	ModelName     string   `yaml:"model_name"`
	CtxImportPath string   `yaml:"ctx_import"`
	RPCs          []RPC    `yaml:"rpcs"`
	Notifies      []Notify `yaml:"notifies"`
}

func LoadDir(dir string) ([]Module, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var modules []Module
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		moduleName := strings.TrimSuffix(entry.Name(), ".yaml")
		if !moduleNamePattern.MatchString(moduleName) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidModuleName, entry.Name())
		}
		module, err := loadFile(filepath.Join(dir, entry.Name()), moduleName)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})
	return modules, nil
}

func loadFile(path string, moduleName string) (Module, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Module{}, err
	}
	var def fileDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		return Module{}, err
	}
	if def.Version != Version {
		return Module{}, fmt.Errorf("%w: %s version %d", ErrUnsupportedVersion, path, def.Version)
	}
	if def.ModelName != moduleName {
		return Module{}, fmt.Errorf("%w: %s model_name %q want %q", ErrInvalidModuleName, path, def.ModelName, moduleName)
	}
	for i := range def.RPCs {
		if def.RPCs[i].CtxImportPath == "" {
			def.RPCs[i].CtxImportPath = def.CtxImportPath
		}
	}
	for i := range def.Notifies {
		if def.Notifies[i].CtxImportPath == "" {
			def.Notifies[i].CtxImportPath = def.CtxImportPath
		}
	}
	return Module{
		Name:          def.ModelName,
		Version:       def.Version,
		CtxImportPath: def.CtxImportPath,
		RPCs:          def.RPCs,
		Notifies:      def.Notifies,
	}, nil
}
