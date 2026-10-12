package i18n

import "testing"

var allSimpleKeys = []Key{
	CommandStart,

	Start,
	NoAccess,
	Unrecognized,
	FeatureUnavailable,
	FirstPage,
	LastPage,

	BtnBack,
	BtnUse,
	BtnEdit,
	BtnSkip,
	BtnDelete,

	OwnerOnly,
	NoUsername,

	BtnOpenMenu,
	BtnNewPosts,
	BtnManageChannels,
	BtnManageAccess,

	BootstrapSuccess,
	BootstrapAlready,
	BootstrapNoOwner,

	PostStatusNew,
	PostStatusSkipped,
	PostStatusScheduled,
	PostStatusSending,
	PostStatusSent,
	PostStatusError,
	PostStatusArchived,
	PostStatusErrorSending,

	FetchMenu,
	FetchChannelsMenu,

	BtnFetchChannels,
	BtnFetchLatest,

	PostListHeader,
	PostListEntry,
	PostDetail,
	BtnSelectPost,

	ScheduledAt,
	ScheduleSkipped,

	ScheduleCustomPrompt,
	ScheduleCustomPlaceholder,
	ScheduleCustomInvalid,
	ScheduleCustomPast,

	ScheduleBtn30m,
	ScheduleBtn1hr,
	ScheduleBtn3hr,
	ScheduleBtn6hr,
	ScheduleBtn12hr,
	ScheduleBtnCustom,

	EditPrompt,
	EditPlaceholder,

	ChannelsMenu,
	ChannelsListHeader,
	ChannelsListEntry,
	ChannelStatusActive,
	ChannelStatusOff,
	BtnChannelsList,
	BtnChannelAdd,
	ChannelAddPrompt,
	ChannelAddHolder,
	ChannelAddInvalid,
	ChannelAddNotFound,
	ChannelAddNotChannel,
	ChannelAddDuplicate,
	ChannelAddSuccess,
	ChannelDeletePrompt,
	ChannelDeleteBlocked,

	AccessMenu,
	AccessListHeader,
	AccessListEntry,
	UserRoleOwner,
	UserRoleAdmin,
	BtnAccessList,
	BtnUserAdd,
	UserAddPrompt,
	UserAddHolder,
	UserAddInvalid,
	UserAddNeedID,
	UserAddDuplicate,
	UserAddSuccess,
	UserDeletePrompt,
	UserDeleteOwnerForbidden,
}

var allPluralKeys = []PluralKey{
	ChannelNewCount,
}

func TestLocalesCompleteness(t *testing.T) {
	for lang := range tags {
		ld := locales[lang]
		for _, key := range allSimpleKeys {
			if _, ok := ld.Simple[key]; !ok {
				t.Errorf("locale %q missing simple key %q", lang, key)
			}
		}

		for _, key := range allPluralKeys {
			forms, ok := ld.Plural[key]
			if !ok {
				t.Errorf("locale %q missing plural key %q", lang, key)
			}

			if _, ok := forms[PluralFormOther]; !ok {
				t.Errorf("locale %q plural key %q missing required %q form", lang, key, PluralFormOther)
			}
		}
	}
}
