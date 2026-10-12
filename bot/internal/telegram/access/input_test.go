package access

import "testing"

func TestParseUserRef(t *testing.T) {
	cases := []struct {
		input    string
		wantKind userRefKind
		wantID   int64
		wantName string
	}{
		{"123456789", refID, 123456789, ""},
		{"  42 ", refID, 42, ""},
		{"@durov", refUsername, 0, "durov"},
		{"@Durov", refUsername, 0, "durov"},
		{"durov", refUsername, 0, "durov"},
		{"my_chat_1", refUsername, 0, "my_chat_1"},

		{"", refInvalid, 0, ""},
		{"   ", refInvalid, 0, ""},
		{"0", refInvalid, 0, ""},
		{"-5", refInvalid, 0, ""},
		{"abc", refInvalid, 0, ""},
		{"abcd", refInvalid, 0, ""},                              // too short for a username
		{"@with space", refInvalid, 0, ""},                       // not a username
		{"кириллица", refInvalid, 0, ""},                         // not a username
		{"a12345678901234567890123456789012", refInvalid, 0, ""}, // too long
	}

	for _, tc := range cases {
		kind, id, name := ParseUserRef(tc.input)
		if kind != tc.wantKind || id != tc.wantID || name != tc.wantName {
			t.Errorf(
				"ParseUserRef(%q) = (%v, %d, %q), want (%v, %d, %q)",
				tc.input, kind, id, name, tc.wantKind, tc.wantID, tc.wantName,
			)
		}
	}
}
