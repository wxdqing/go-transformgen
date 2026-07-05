package chat

import bootstrap "gitee.com/wxdqing/fx-bootstrap"

type Provider struct {
	bootstrap.NopHook
}

func (Provider) Register() any {
	return NewModule
}
