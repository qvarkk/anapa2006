package i18n

import "golang.org/x/text/feature/plural"

type PluralForm string

const (
	PluralFormOther PluralForm = "other"
	PluralFormZero  PluralForm = "zero"
	PluralFormOne   PluralForm = "one"
	PluralFormTwo   PluralForm = "two"
	PluralFormFew   PluralForm = "few"
	PluralFormMany  PluralForm = "many"
)

func formToPluralForm(form plural.Form) PluralForm {
	switch form {
	case plural.Other:
		return PluralFormOther
	case plural.Zero:
		return PluralFormZero
	case plural.One:
		return PluralFormOne
	case plural.Two:
		return PluralFormTwo
	case plural.Few:
		return PluralFormFew
	case plural.Many:
		return PluralFormMany
	default:
		return PluralFormOther
	}
}
