// Package moscow converts between an instant and Moscow wall-clock time.
//
// Moscow has been on a fixed UTC+3 offset without daylight saving since 2014,
// so the zone is a fixed offset. That also keeps the bot independent of tzdata
// being present in the container image.
package moscow

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Location is Moscow standard time (UTC+3, no DST).
var Location = time.FixedZone("MSK", 3*60*60)

// ErrFormat is returned when the input matches no accepted date/time layout.
var ErrFormat = errors.New("moscow: unrecognized date/time format")

var (
	timeOnlyRe = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)
	dateTimeRe = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})(?:\.(\d{2}|\d{4}))?\s+(\d{1,2}):(\d{2})$`)
)

// Format renders an instant as Moscow date and time.
func Format(t time.Time) string {
	return t.In(Location).Format("02.01.2006 15:04")
}

// Parse interprets input as Moscow wall-clock time. A missing date means today
// in Moscow, a missing year means the current Moscow year. Accepted forms:
//
//	HH:MM
//	DD.MM HH:MM
//	DD.MM.YYYY HH:MM
func Parse(input string, now time.Time) (time.Time, error) {
	s := normalize(input)
	nowMSK := now.In(Location)

	if m := timeOnlyRe.FindStringSubmatch(s); m != nil {
		return build(nowMSK.Year(), int(nowMSK.Month()), nowMSK.Day(), atoi(m[1]), atoi(m[2]))
	}

	m := dateTimeRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, ErrFormat
	}

	year := nowMSK.Year()
	if m[3] != "" {
		year = atoi(m[3])
		if year < 100 {
			year += 2000
		}
	}

	return build(year, atoi(m[2]), atoi(m[1]), atoi(m[4]), atoi(m[5]))
}

// build validates the wall-clock fields, since time.Date silently normalizes
// out-of-range values such as day 32 or minute 99.
func build(year, month, day, hour, minute int) (time.Time, error) {
	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, Location)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day ||
		t.Hour() != hour || t.Minute() != minute {
		return time.Time{}, ErrFormat
	}
	return t, nil
}

// normalize makes common typing variations acceptable: extra spaces, a trailing
// dot, a comma between date and time, and the Russian "в" separator.
func normalize(input string) string {
	s := strings.TrimSpace(input)
	s = strings.TrimSuffix(s, ".")
	s = strings.ReplaceAll(s, ",", " ")
	s = strings.ReplaceAll(s, " в ", " ")
	return strings.Join(strings.Fields(s), " ")
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
