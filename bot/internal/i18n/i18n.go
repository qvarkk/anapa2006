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
	Unscheduled        Key = "unscheduled"
	OperationSuccess   Key = "operation_success"

	// Common buttons
	BtnBack   Key = "btn_back"
	BtnUse    Key = "btn_use"
	BtnEdit   Key = "btn_edit"
	BtnSkip   Key = "btn_skip"
	BtnDelete Key = "btn_delete"

	// Common messages
	OwnerOnly  Key = "owner_only"
	NoUsername Key = "no_username"

	// Start menu buttons
	BtnOpenMenu       Key = "btn_open_menu"
	BtnNewPosts       Key = "btn_new_posts"
	BtnManageChannels Key = "btn_manage_channels"
	BtnManageAccess   Key = "btn_manage_access"

	// Bootstrap
	BootstrapSuccess Key = "bootstrap_success"
	BootstrapAlready Key = "bootstrap_already"
	BootstrapNoOwner Key = "bootstrap_no_owner"

	// Posts statuses
	PostStatusNew          Key = "post_status_new"
	PostStatusSkipped      Key = "post_status_skipped"
	PostStatusScheduled    Key = "post_status_scheduled"
	PostStatusSending      Key = "post_status_sending"
	PostStatusSent         Key = "post_status_sent"
	PostStatusError        Key = "post_status_error"
	PostStatusArchived     Key = "post_status_archived"
	PostStatusErrorSending Key = "post_status_error_sending"

	// Post buttons
	BtnPostShow      Key = "btn_post_show"
	BtnPostHide      Key = "btn_post_hide"
	BtnPostDelete    Key = "btn_post_delete"
	BtnPostArchive   Key = "btn_post_archive"
	BtnPostUnarchive Key = "btn_post_unarchive"

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
	BtnSelectPost  Key = "btn_select_post"

	// Schedule headers
	ScheduledAt     Key = "scheduled_at"
	ScheduleSkipped Key = "schedule_skipped"

	// Custom schedule input
	ScheduleCustomPrompt      Key = "schedule_custom_prompt"
	ScheduleCustomPlaceholder Key = "schedule_custom_placeholder"
	ScheduleCustomInvalid     Key = "schedule_custom_invalid"
	ScheduleCustomPast        Key = "schedule_custom_past"

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

	// Drafts
	DraftsMenu  Key = "drafts_menu"
	DraftsList  Key = "drafts_list"
	DraftDetail Key = "draft_detail"

	// Draft actions
	DraftCreated Key = "draft_created"

	// Draft prompts
	PromptDraftEditText  Key = "promps_draft_edit_text"
	PromptDraftEditMedia Key = "promps_draft_edit_media"
	PromptDraftQueue     Key = "promps_draft_queue"
	PromptDraftDequeue   Key = "promps_draft_dequeue"
	PromptDraftDelete    Key = "promps_draft_delete"

	// Draft prompts buttons
	BtnPromptDraftDequeue Key = "btn_prompt_draft_dequeue"
	BtnPromptDraftDelete  Key = "btn_prompt_draft_delete"

	// Draft buttons
	BtnDraftsMenu     Key = "btn_drafts_menu"
	BtnDraftsList     Key = "btn_drafts_list"
	BtnDraftDetail    Key = "btn_draft_detail"
	BtnDraftView      Key = "btn_draft_view"
	BtnDraftCreate    Key = "btn_draft_create"
	BtnDraftEditText  Key = "btn_draft_edit_text"
	BtnDraftEditMedia Key = "btn_draft_edit_media"
	BtnDraftRequeue   Key = "btn_draft_requeue"
	BtnDraftQueue     Key = "btn_draft_queue"
	BtnDraftDequeue   Key = "btn_draft_dequeue"
	BtnDraftDelete    Key = "btn_draft_delete"
	BtnDraftShow      Key = "btn_draft_show"

	// Channel management
	ChannelsMenu         Key = "channels_menu"
	ChannelsListHeader   Key = "channels_list_header"
	ChannelsListEntry    Key = "channels_list_entry"
	ChannelStatusActive  Key = "channel_status_active"
	ChannelStatusOff     Key = "channel_status_off"
	ChannelStats         Key = "channel_stats"
	ChannelDetail        Key = "channel_detail"
	BtnChannelsList      Key = "btn_channels_list"
	BtnChannelAdd        Key = "btn_channel_add"
	BtnChannelEnable     Key = "btn_channel_enable"
	BtnChannelDisable    Key = "btn_channel_disable"
	ChannelAddPrompt     Key = "channel_add_prompt"
	ChannelAddHolder     Key = "channel_add_placeholder"
	ChannelAddInvalid    Key = "channel_add_invalid"
	ChannelAddNotFound   Key = "channel_add_not_found"
	ChannelAddNotChannel Key = "channel_add_not_channel"
	ChannelAddDuplicate  Key = "channel_add_duplicate"
	ChannelAddSuccess    Key = "channel_add_success"
	ChannelDeletePrompt  Key = "channel_delete_prompt"
	ChannelDeleteBlocked Key = "channel_delete_blocked"

	// Access management
	AccessMenu               Key = "access_menu"
	AccessListHeader         Key = "access_list_header"
	AccessListEntry          Key = "access_list_entry"
	UserRoleOwner            Key = "user_role_owner"
	UserRoleAdmin            Key = "user_role_admin"
	BtnAccessList            Key = "btn_access_list"
	BtnUserAdd               Key = "btn_user_add"
	UserAddPrompt            Key = "user_add_prompt"
	UserAddHolder            Key = "user_add_placeholder"
	UserAddInvalid           Key = "user_add_invalid"
	UserAddNeedID            Key = "user_add_need_id"
	UserAddNotUser           Key = "user_add_not_user"
	UserAddDuplicate         Key = "user_add_duplicate"
	UserAddSuccess           Key = "user_add_success"
	UserDeletePrompt         Key = "user_delete_prompt"
	UserDeleteOwnerForbidden Key = "user_delete_owner_forbidden"

	// Target channels (publishing destinations)
	TargetsMenu           Key = "targets_menu"
	TargetsListHeader     Key = "targets_list_header"
	TargetsListEntry      Key = "targets_list_entry"
	BtnTargetsList        Key = "btn_targets_list"
	BtnTargetsMenu        Key = "btn_targets_menu"
	BtnTargetAdd          Key = "btn_target_add"
	BtnTargetDelete       Key = "btn_target_delete"
	TargetAddPrompt       Key = "target_add_prompt"
	TargetAddHolder       Key = "target_add_placeholder"
	TargetAddInvalid      Key = "target_add_invalid"
	TargetAddNotFound     Key = "target_add_not_found"
	TargetAddNotChannel   Key = "target_add_not_channel"
	TargetAddDuplicate    Key = "target_add_duplicate"
	TargetAddInconclusive Key = "target_add_inconclusive"
	TargetAddSuccess      Key = "target_add_success"
	TargetAddNoAdmin      Key = "target_add_no_admin"
	TargetDeletePrompt    Key = "target_delete_prompt"

	// Draft list states
	DraftsListActiveHeader Key = "drafts_list_active_header"
	DraftsListPostedHeader Key = "drafts_list_posted_header"
	DraftsListFailedHeader Key = "drafts_list_failed_header"
	DraftListPostedEntry   Key = "draft_list_posted_entry"
	DraftListFailedEntry   Key = "draft_list_failed_entry"
	DraftErrorUnknown      Key = "draft_error_unknown"
	BtnDraftsActive        Key = "btn_drafts_active"
	BtnDraftsPosted        Key = "btn_drafts_posted"
	BtnDraftsFailed        Key = "btn_drafts_failed"

	// Draft detail states
	DraftDetailActive    Key = "draft_detail_active"
	DraftDetailPosted    Key = "draft_detail_posted"
	DraftDetailFailed    Key = "draft_detail_failed"
	DraftTargetWarning   Key = "draft_target_warning"
	BtnDraftTarget       Key = "btn_draft_target"
	BtnDraftSelectTarget Key = "btn_draft_select_target"
	TargetNotSelected    Key = "target_not_selected"
	BtnOpenPosted        Key = "btn_open_posted"
	DraftNeedTarget      Key = "draft_need_target"
	DraftTargetPrompt    Key = "draft_target_prompt"
	BtnDraftTargetClear  Key = "btn_draft_target_clear"

	// Draft media editor
	DraftMediaMenu          Key = "draft_media_menu"
	DraftMediaEntry         Key = "draft_media_entry"
	DraftMediaDetail        Key = "draft_media_detail"
	BtnDraftMediaItem       Key = "btn_draft_media_item"
	BtnMediaAdd             Key = "btn_media_add"
	BtnMediaSubstitute      Key = "btn_media_substitute"
	BtnMediaDelete          Key = "btn_media_delete"
	BtnMediaBack            Key = "btn_media_back"
	MediaSubstitutePrompt   Key = "media_substitute_prompt"
	MediaPlaceholder        Key = "media_placeholder"
	MediaAddPositionPrompt  Key = "media_add_position_prompt"
	MediaAddPositionHolder  Key = "media_add_position_placeholder"
	MediaAddInvalidPosition Key = "media_add_invalid_position"
	MediaAddPrompt          Key = "media_add_prompt"
	MediaLimitReached       Key = "media_limit_reached"
	MediaUnsupported        Key = "media_unsupported"
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
