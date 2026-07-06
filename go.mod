module github.com/wxdqing/go-transformgen

go 1.26.3

require (
	gitee.com/wxdqing/fx-bootstrap v0.0.0
	go.uber.org/fx v1.24.0
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace gitee.com/wxdqing/fx-bootstrap v0.0.0 => ../../../libs/fx-bootstrap
