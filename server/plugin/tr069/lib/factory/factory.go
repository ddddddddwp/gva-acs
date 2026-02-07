package factory

import (
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/lib/builder"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/lib/parser"
)

func NewParser() *parser.Parser {
	return parser.NewParser()
}

func NewBuilder() *builder.Builder {
	return builder.NewBuilder()
}
