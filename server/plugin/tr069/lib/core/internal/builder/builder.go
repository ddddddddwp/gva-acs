package builder

import (
	"context"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
)

type Builder struct{}

func New(opts ...tr069.Option) *Builder {
	return &Builder{}
}

func (b *Builder) BuildMessage(ctx context.Context, msg *tr069.Message) ([]byte, error) {
	return []byte("<mock>response</mock>"), nil
}
