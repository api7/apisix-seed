package tools

import (
	"e2e/tools/common"
	"e2e/tools/regcenter"
)

type IRegCenter interface {
	Online(*common.Node) error
	Offline(*common.Node) error
	Clean() error
	//Query()
}

func NewIRegCenter(name string) IRegCenter {
	switch name {
	case "nacos":
		return regcenter.NewNacos()
	}
	return nil
}
