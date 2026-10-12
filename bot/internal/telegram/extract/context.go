package extract

import (
	"context"
	"qq/anapa2006/internal/i18n"
)

// ctxKey is unexported so values set through this package cannot collide with
// keys set by other packages.
type ctxKey int

const (
	ctxKeyLang ctxKey = iota
	ctxKeyOwner
)

// WithLang stores the language of the acting user.
func WithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, ctxKeyLang, lang)
}

// WithOwner records whether the acting user is the bot owner.
func WithOwner(ctx context.Context, owner bool) context.Context {
	return context.WithValue(ctx, ctxKeyOwner, owner)
}

func Lang(ctx context.Context) i18n.Lang {
	if v, ok := ctx.Value(ctxKeyLang).(string); ok {
		return i18n.ToLang(v)
	}
	return i18n.DefaultLang
}

// IsOwner reports whether the acting user is the configured bot owner.
func IsOwner(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyOwner).(bool)
	return v
}
