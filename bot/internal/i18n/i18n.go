package i18n

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

//go:embed locales/*.yaml
var localeFS embed.FS

type localeData struct {
	Simple map[Key]string                      `yaml:"simple"`
	Plural map[PluralKey]map[PluralForm]string `yaml:"plural"`
}

type Lang string
type Key string
type PluralKey string

// !!!
// WHEN ADDING A NEW KEY
// DON'T FORGET TO COMPLY TO THE POOPY SPAGHETTI CODE
// AND ADD IT TO THE TEST
// !!!
const (
	KeyStart    Key = "start"
	KeyNoAccess Key = "no_access"
)

// !!!
// WHEN ADDING A NEW KEY
// DON'T FORGET TO COMPLY TO THE POOPY SPAGHETTI CODE
// AND ADD IT TO THE TEST
// !!!
const (
	KeyNewPostsCount PluralKey = "new_posts_count"
)

const (
	LangRU Lang = "ru"
)

const DefaultLang = LangRU

var (
	locales = map[Lang]localeData{}
	tags    = map[Lang]language.Tag{
		LangRU: language.Russian,
	}
)

func init() {
	for lang := range tags {
		data, err := localeFS.ReadFile("locales/" + string(lang) + ".yaml")
		if err != nil {
			panic(fmt.Sprintf("i18n: missing locale file for %q: %v", lang, err))
		}
		var ld localeData
		if err := yaml.Unmarshal(data, &ld); err != nil {
			panic(fmt.Sprintf("i18n: bad yaml for %q: %v", lang, err))
		}
		locales[lang] = ld
	}
}

// Returns a plain (non-plural) message for given language and key.
func T(lang Lang, key Key, args ...any) string {
	ld, ok := locales[lang]
	if !ok {
		slog.LogAttrs(
			context.Background(),
			slog.LevelDebug,
			"i18n: missing locale data",
			slog.String("locale", string(lang)),
		)
		return string(key)
	}

	tmpl, ok := ld.Simple[key]
	if !ok {
		slog.LogAttrs(
			context.Background(),
			slog.LevelDebug,
			"i18n: missing simple key",
			slog.String("locale", string(lang)),
			slog.String("key", string(key)),
		)
		return string(key)
	}

	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}

// Returns a pluralized message for given language, key, and count n.
// The chosen n is inserted into the message at position 0 automatically.
func TN(lang Lang, key PluralKey, n int, args ...any) string {
	ld, ok := locales[lang]
	if !ok {
		slog.LogAttrs(
			context.Background(),
			slog.LevelDebug,
			"i18n: missing locale data",
			slog.String("locale", string(lang)),
		)
		return string(key)
	}

	forms, ok := ld.Plural[key]
	if !ok {
		slog.LogAttrs(
			context.Background(),
			slog.LevelDebug,
			"i18n: missing plural key",
			slog.String("locale", string(lang)),
			slog.String("key", string(key)),
		)
		return string(key)
	}

	tag, ok := tags[lang]
	if !ok {
		tag = language.Russian
	}
	form := plural.Cardinal.MatchPlural(tag, n, 0, 0, 0, 0)
	pluralKey := formToPluralForm(form)

	tmpl, ok := forms[pluralKey]
	if !ok {
		tmpl = forms[PluralFormOther]
	}

	allArgs := append([]any{n}, args...)
	return fmt.Sprintf(tmpl, allArgs...)
}

func ToLang(v string) Lang {
	switch v {
	case string(LangRU):
		return LangRU
	default:
		return DefaultLang
	}
}
