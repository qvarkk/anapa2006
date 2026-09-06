package telegram

import (
	"context"
	"qq/anapa2006/internal/i18n"
)

type ctxKey string

const (
	ctxKeyLang ctxKey = "lang"
)

func LangFromContext(ctx context.Context) i18n.Lang {
	if v, ok := ctx.Value(ctxKeyLang).(string); ok {
		return i18n.ToLang(v)
	}
	return i18n.DefaultLang
}
