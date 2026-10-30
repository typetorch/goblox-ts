package diagnostics

import (
	"os"

	"golang.org/x/term"
)

// ColorEnabled mirrors kleur's `$.enabled`: roblox-ts bakes kleur colors into diagnostic messages, so
// the message text depends on it.
var ColorEnabled = func() bool {
	_, noColor := os.LookupEnv("NO_COLOR")
	if _, ok := os.LookupEnv("NODE_DISABLE_COLORS"); ok || noColor || os.Getenv("TERM") == "dumb" {
		return false
	}
	if force, ok := os.LookupEnv("FORCE_COLOR"); ok && force != "0" {
		return true
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}()

func color(openCode, closeCode string, text string) string {
	if !ColorEnabled {
		return text
	}
	return "\x1b[" + openCode + "m" + text + "\x1b[" + closeCode + "m"
}

// Yellow is kleur.yellow.
func Yellow(text string) string { return color("33", "39", text) }

// Grey is kleur.grey.
func Grey(text string) string { return color("90", "39", text) }
