package i18n

import "testing"

var allSimpleKeys = []Key{
	KeyCommandStart,

	KeyStart,
	KeyNoAccess,
	KeyUnrecognized,
	KeyFeatureUnavailable,

	KeyBtnBack,
	KeyBtnSelectPost,
	KeyBtnUse,
	KeyBtnEdit,
	KeyBtnSkip,

	KeyBtnOpenMenu,
	KeyBtnNewPosts,
	KeyBtnScheduled,

	KeyPostStatusNew,
	KeyPostStatusSkipped,
	KeyPostStatusReviewing,
	KeyPostStatusArchived,

	KeyListFetchedMenuTitle,
	KeyListFetchedChannelTitle,

	KeyBtnFetchedByChannel,
	KeyBtnFetchedLatest,

	KeyPostListHeader,
	KeyPostListEntry,
	KeyPostDetail,
}

var allPluralKeys = []PluralKey{
	KeyChannelNewCount,
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
