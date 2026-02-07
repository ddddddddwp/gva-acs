package factory

import (
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/internal/parser"
	"github.com/ddddddddwp/tr069-core-only/internal/builder"
)

func NewParser(opts ...tr069.Option) tr069.Parser {
	return parser.New(opts...)
}

func NewBuilder(opts ...tr069.Option) tr069.Builder {
	return builder.New(opts...)
}
