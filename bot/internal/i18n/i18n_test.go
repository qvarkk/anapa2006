package i18n

import "testing"

var allSimpleKeys = []Key{
	KeyStart,
	KeyNoAccess,
}

var allPluralKeys = []PluralKey{
	KeyNewPostsCount,
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
