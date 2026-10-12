package domain

import "testing"

func TestNormalizeUsername(t *testing.T) {
	cases := map[string]string{
		"durov":     "durov",
		"  Durov ":  "durov",
		"@Durov":    "durov",
		"@ durov":   "durov",
		"my_chat_1": "my_chat_1",
		"":          "",
	}
	for in, want := range cases {
		if got := NormalizeUsername(in); got != want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidUsername(t *testing.T) {
	valid := []string{"durov", "my_chat_1", "abcde", "a1234567890123456789012345678901"}
	for _, in := range valid {
		if !ValidUsername(in) {
			t.Errorf("ValidUsername(%q) = false, want true", in)
		}
	}

	invalid := []string{
		"", "abcd", // too short
		"a12345678901234567890123456789012", // 33 chars, too long
		"with space", "кириллица", "with-dash", "@durov",
	}
	for _, in := range invalid {
		if ValidUsername(in) {
			t.Errorf("ValidUsername(%q) = true, want false", in)
		}
	}
}
