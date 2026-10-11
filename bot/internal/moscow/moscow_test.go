package moscow

import (
	"errors"
	"testing"
	"time"
)

func mustNow(t *testing.T) time.Time {
	t.Helper()
	// 2026-10-11 12:00 MSK == 09:00 UTC
	return time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
}

func TestParse(t *testing.T) {
	now := mustNow(t)

	ok := []struct {
		input string
		want  time.Time
	}{
		{"14:30", time.Date(2026, 10, 11, 14, 30, 0, 0, Location)},
		{"9:07", time.Date(2026, 10, 11, 9, 7, 0, 0, Location)},
		{"  25.12   09:05  ", time.Date(2026, 12, 25, 9, 5, 0, 0, Location)},
		{"5.3 8:07", time.Date(2026, 3, 5, 8, 7, 0, 0, Location)},
		{"01.02.2027 00:00", time.Date(2027, 2, 1, 0, 0, 0, 0, Location)},
		{"01.02.27 23:59", time.Date(2027, 2, 1, 23, 59, 0, 0, Location)},
		{"25.12.2026, 14:30", time.Date(2026, 12, 25, 14, 30, 0, 0, Location)},
		{"25.12 в 14:30", time.Date(2026, 12, 25, 14, 30, 0, 0, Location)},
		{"25.12 14:30.", time.Date(2026, 12, 25, 14, 30, 0, 0, Location)},
	}

	for _, tc := range ok {
		got, err := Parse(tc.input, now)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("Parse(%q) = %s, want %s", tc.input, got, tc.want)
		}
		if got.Location() != Location {
			t.Errorf("Parse(%q) location = %v, want Moscow", tc.input, got.Location())
		}
	}

	bad := []string{"", "   ", "abc", "14", "14:5", "25.12", "32.12 10:00", "10.13 10:00", "10:99", "25.12 24:00"}
	for _, input := range bad {
		if _, err := Parse(input, now); !errors.Is(err, ErrFormat) {
			t.Errorf("Parse(%q) error = %v, want ErrFormat", input, err)
		}
	}
}

func TestFormat(t *testing.T) {
	utc := time.Date(2026, 10, 11, 9, 30, 0, 0, time.UTC)
	if got, want := Format(utc), "11.10.2026 12:30"; got != want {
		t.Errorf("Format(UTC) = %q, want %q", got, want)
	}
}
