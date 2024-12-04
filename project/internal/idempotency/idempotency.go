package idempotency

import (
	"context"
	"github.com/google/uuid"
)

type ctxKey struct{}

func WithKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxKey{}, key)
}

func GetKey(ctx context.Context) string {
	key, ok := ctx.Value(ctxKey{}).(string)
	if !ok {
		return uuid.NewString()
	}

	return key
}
