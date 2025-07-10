package db

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

type FilteredTracer struct {
	inner     pgx.QueryTracer
	skipTable string
}

// skipCtxKey is a unique type to store skip flag in context
type skipCtxKey struct{}

func (t *FilteredTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	// Normalize to lowercase before checking to make comparison case-insensitive.
	if strings.Contains(strings.ToLower(data.SQL), strings.ToLower(t.skipTable)) {
		// Mark this context as skipped so that TraceQueryEnd can also skip safely.
		return context.WithValue(ctx, skipCtxKey{}, true)
	}

	return t.inner.TraceQueryStart(ctx, conn, data)
}

func (t *FilteredTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	// If TraceQueryStart marked this context as skipped, do nothing.
	if ctx.Value(skipCtxKey{}) != nil {
		return
	}

	// As an extra safeguard (e.g. if a different context is passed), also check
	// the command tag for the skipTable string. This ensures symmetry even if
	// the context was not propagated for some reason.
	if strings.Contains(strings.ToLower(data.CommandTag.String()), strings.ToLower(t.skipTable)) {
		return
	}

	t.inner.TraceQueryEnd(ctx, conn, data)
}
