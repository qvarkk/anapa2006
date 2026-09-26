package extract

import (
	"context"
	"qq/anapa2006/internal/i18n"
)

const (
	CtxKeyLang = "lang"
)

func Lang(ctx context.Context) i18n.Lang {
	if v, ok := ctx.Value(CtxKeyLang).(string); ok {
		return i18n.ToLang(v)
	}
	return i18n.DefaultLang
}
