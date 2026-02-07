package builder

import (
	"context"
	"fmt"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/lib/parser"
)

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) BuildMessage(ctx context.Context, msg *parser.Message) ([]byte, error) {
	// Mock implementation
	return []byte(fmt.Sprintf(`<soap:Envelope><soap:Body><cwmp:%sResponse><MaxEnvelopes>1</MaxEnvelopes></cwmp:%sResponse></soap:Body></soap:Envelope>`, msg.Method, msg.Method)), nil
}
