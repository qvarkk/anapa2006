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
	// Command desciptions
	KeyCommandStart Key = "start_command_description"

	// Common titles
	KeyStart              Key = "start"
	KeyNoAccess           Key = "no_access"
	KeyUnrecognized       Key = "unrecognized_command"
	KeyFeatureUnavailable Key = "feature_unavailable"

	// Common buttons
	KeyBtnBack       Key = "btn_back"
	KeyBtnSelectPost Key = "btn_select_post"
	KeyBtnUse        Key = "btn_use"
	KeyBtnEdit       Key = "btn_edit"
	KeyBtnSkip       Key = "btn_skip"

	// Start menu buttons
	KeyBtnOpenMenu  Key = "btn_open_menu"
	KeyBtnNewPosts  Key = "btn_new_posts"
	KeyBtnScheduled Key = "btn_scheduled"

	// Posts statuses
	KeyPostStatusNew       Key = "post_status_new"
	KeyPostStatusSkipped   Key = "post_status_skipped"
	KeyPostStatusReviewing Key = "post_status_reviewing"
	KeyPostStatusArchived  Key = "post_status_archived"

	// List fetched titles
	KeyListFetchedMenuTitle    Key = "list_fetched_menu_title"
	KeyListFetchedChannelTitle Key = "list_fetched_channel_title"

	// List fetched buttons
	KeyBtnFetchedByChannel Key = "btn_fetched_by_channel"
	KeyBtnFetchedLatest    Key = "btn_fetched_latest"

	// Post lists
	KeyPostListHeader Key = "post_list_header"
	KeyPostListEntry  Key = "post_list_entry"
	KeyPostDetail     Key = "post_detail"
)

// !!!
// WHEN ADDING A NEW KEY
// DON'T FORGET TO COMPLY TO THE POOPY SPAGHETTI CODE
// AND ADD IT TO THE TEST
// !!!
const (
	KeyChannelNewCount PluralKey = "channel_new_count"
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
