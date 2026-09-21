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
	CommandStart Key = "start_command_description"

	// Common titles
	Start              Key = "start"
	NoAccess           Key = "no_access"
	Unrecognized       Key = "unrecognized_command"
	FeatureUnavailable Key = "feature_unavailable"
	FirstPage          Key = "first_page"
	LastPage           Key = "last_page"

	// Common buttons
	BtnBack       Key = "btn_back"
	BtnSelectPost Key = "btn_select_post"
	BtnUse        Key = "btn_use"
	BtnEdit       Key = "btn_edit"
	BtnSkip       Key = "btn_skip"

	// Start menu buttons
	BtnOpenMenu  Key = "btn_open_menu"
	BtnNewPosts  Key = "btn_new_posts"
	BtnScheduled Key = "btn_scheduled"

	// Posts statuses
	PostStatusNew       Key = "post_status_new"
	PostStatusSkipped   Key = "post_status_skipped"
	PostStatusScheduled Key = "post_status_scheduled"
	PostStatusSent      Key = "post_status_sent"

	// List fetched titles
	FetchMenu         Key = "list_fetched_menu_title"
	FetchChannelsMenu Key = "list_fetched_channel_title"

	// List fetched buttons
	BtnFetchChannels Key = "btn_fetched_by_channel"
	BtnFetchLatest   Key = "btn_fetched_latest"

	// Post lists
	PostListHeader Key = "post_list_header"
	PostListEntry  Key = "post_list_entry"
	PostDetail     Key = "post_detail"

	// Schedule headers
	SchedulePrompt  Key = "schedule_prompt"
	ScheduledAt     Key = "scheduled_at"
	ScheduleSkipped Key = "schedule_skipped"

	// Schedule btns
	ScheduleBtn30m    Key = "schedule_btn_30m"
	ScheduleBtn1hr    Key = "schedule_btn_1hr"
	ScheduleBtn3hr    Key = "schedule_btn_3hr"
	ScheduleBtn6hr    Key = "schedule_btn_6hr"
	ScheduleBtn12hr   Key = "schedule_btn_12hr"
	ScheduleBtnCustom Key = "schedule_btn_custom"

	// Schedule edit
	EditPrompt      Key = "edit_prompt"
	EditPlaceholder Key = "edit_placeholder"
)

// !!!
// WHEN ADDING A NEW KEY
// DON'T FORGET TO COMPLY TO THE POOPY SPAGHETTI CODE
// AND ADD IT TO THE TEST
// !!!
const (
	ChannelNewCount PluralKey = "channel_new_count"
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
