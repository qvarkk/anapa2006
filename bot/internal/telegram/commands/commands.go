package commands

import "strings"

const (
	Start     = "/start"
	Bootstrap = "/bootstrap"
)

// IsBootstrap matches the bootstrap command, including the "/bootstrap@botname"
// form Telegram uses for commands sent in groups.
func IsBootstrap(text string) bool {
	return text == Bootstrap || strings.HasPrefix(text, Bootstrap+"@")
}
