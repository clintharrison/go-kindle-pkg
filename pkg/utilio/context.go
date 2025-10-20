package utilio

import (
	"context"
	"io"
)

type ContextReader struct {
	ctx context.Context //nolint:containedctx
	r   io.Reader
}

func NewContextReader(ctx context.Context, r io.Reader) *ContextReader {
	return &ContextReader{ctx, r}
}

func (r *ContextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err() //nolint:wrapcheck
	default:
		return r.r.Read(p) //nolint:wrapcheck
	}
}
