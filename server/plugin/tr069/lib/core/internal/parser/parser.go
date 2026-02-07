package parser

import (
	"context"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
)

type Parser struct{}

func New(opts ...tr069.Option) *Parser {
	return &Parser{}
}

func (p *Parser) ParseMessage(ctx context.Context, data []byte) (*tr069.Message, error) {
	return &tr069.Message{
		ID:     "MOCK_ID",
		Method: "Inform",
		DeviceID: struct{ SerialNumber string }{
			SerialNumber: "MOCK_SN",
		},
	}, nil
}
