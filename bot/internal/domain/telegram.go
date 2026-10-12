package domain

import (
	"regexp"
	"strings"
)

// Telegram usernames and public channel handles share the same shape:
// latin letters, digits and underscores, 5 to 32 characters.
var usernameRe = regexp.MustCompile(`^[a-z0-9_]{5,32}$`)

// NormalizeUsername makes user-typed handles comparable with what Telegram
// reports: surrounding spaces and a leading "@" are dropped, and it is lowered.
func NormalizeUsername(input string) string {
	handle := strings.TrimSpace(input)
	handle = strings.TrimPrefix(handle, "@")
	return strings.ToLower(strings.TrimSpace(handle))
}

// ValidUsername reports whether the normalized handle can be a Telegram handle.
func ValidUsername(handle string) bool {
	return usernameRe.MatchString(handle)
}
