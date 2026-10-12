package access

import "testing"

func TestParseUserRef(t *testing.T) {
	cases := []struct {
		input    string
		wantKind userRefKind
		wantID   int64
	}{
		{"123456789", refID, 123456789},
		{"  42 ", refID, 42},

		{"", refInvalid, 0},
		{"   ", refInvalid, 0},
		{"0", refInvalid, 0},
		{"-5", refInvalid, 0},
		{"abc", refInvalid, 0},
		{"@durov", refInvalid, 0},
		{"durov", refInvalid, 0},
		{"кириллица", refInvalid, 0},
		{"12.5", refInvalid, 0},
	}

	for _, tc := range cases {
		kind, id := ParseUserRef(tc.input)
		if kind != tc.wantKind || id != tc.wantID {
			t.Errorf(
				"ParseUserRef(%q) = (%v, %d), want (%v, %d)",
				tc.input, kind, id, tc.wantKind, tc.wantID,
			)
		}
	}
}
