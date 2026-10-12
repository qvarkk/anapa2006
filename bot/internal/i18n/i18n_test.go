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

	BtnPostUnarchive,

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
	ChannelStats,
	ChannelDetail,
	BtnChannelsList,
	BtnChannelAdd,
	BtnChannelEnable,
	BtnChannelDisable,
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
	UserAddNotUser,
	UserAddDuplicate,
	UserAddSuccess,
	UserDeletePrompt,
	UserDeleteOwnerForbidden,

	TargetsMenu,
	TargetsListHeader,
	TargetsListEntry,
	BtnTargetsList,
	BtnTargetsMenu,
	BtnTargetAdd,
	BtnTargetDelete,
	TargetAddPrompt,
	TargetAddHolder,
	TargetAddInvalid,
	TargetAddNotFound,
	TargetAddNotChannel,
	TargetAddDuplicate,
	TargetAddInconclusive,
	TargetAddSuccess,
	TargetAddNoAdmin,
	TargetDeletePrompt,

	DraftsListActiveHeader,
	DraftsListPostedHeader,
	DraftsListFailedHeader,
	DraftListPostedEntry,
	DraftListFailedEntry,
	DraftErrorUnknown,
	BtnDraftsActive,
	BtnDraftsPosted,
	BtnDraftsFailed,

	DraftDetailActive,
	DraftDetailPosted,
	DraftDetailFailed,
	DraftTargetWarning,
	BtnDraftTarget,
	BtnDraftSelectTarget,
	TargetNotSelected,
	BtnOpenPosted,
	DraftNeedTarget,
	DraftTargetPrompt,
	BtnDraftTargetClear,

	DraftMediaMenu,
	DraftMediaEntry,
	DraftMediaDetail,
	BtnDraftMediaItem,
	BtnMediaAdd,
	BtnMediaSubstitute,
	BtnMediaDelete,
	BtnMediaBack,
	MediaSubstitutePrompt,
	MediaPlaceholder,
	MediaAddPositionPrompt,
	MediaAddPositionHolder,
	MediaAddInvalidPosition,
	MediaAddPrompt,
	MediaLimitReached,
	MediaUnsupported,
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
