package main

import (
	"flag"
	"strings"
)

// normalizeArgs turns rbxtsc's yargs command line into Go flags, and returns the message of the first
// check that yargs would fail (.strict(), .choices()), "" if there is none. yargs accepts `--flag value`
// for booleans, `--kebab-case` names and `--no-flag`, and gives a string option without a value the empty
// string. Single dash flags stay one Go flag if they exist (-verbose), and are yargs's letters otherwise.
func normalizeArgs(flags *flag.FlagSet, args []string) ([]string, string) {
	var result []string
	var unknown []string
	typeWithoutValue := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			unknown = append(unknown, args[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			unknown = append(unknown, arg)
			continue
		}
		rawName, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		// yargs-parser's boolean negation comes first, so --no-include is not --noInclude
		negated := false
		if positive, ok := strings.CutPrefix(rawName, "no-"); ok && strings.HasPrefix(arg, "--") && !hasValue {
			rawName, negated = positive, true
		}
		name := kebabToCamel(rawName)
		// yargs' .help() wins over every check
		if name == "help" || name == "h" {
			return []string{"-h"}, ""
		}
		if flags.Lookup(name) == nil {
			switch {
			case !strings.HasPrefix(arg, "--"):
				unknown = append(unknown, strings.Split(name, "")...)
			case name != rawName:
				unknown = append(unknown, rawName, name)
			default:
				unknown = append(unknown, name)
			}
			// the value of an unknown option is not a positional
			if !hasValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		}
		hasNext := i+1 < len(args)
		switch {
		case negated:
			// a negated string option is the same as not giving it
			if isBoolFlag(flags.Lookup(name)) {
				result = append(result, "--"+name+"=false")
			}
		case hasValue:
			result = append(result, "--"+name+"="+value)
		case isBoolFlag(flags.Lookup(name)):
			if hasNext && (args[i+1] == "true" || args[i+1] == "false") {
				i++
				result = append(result, "--"+name+"="+args[i])
			} else {
				result = append(result, "--"+name)
			}
		case hasNext && !strings.HasPrefix(args[i+1], "-"):
			i++
			result = append(result, "--"+name+"="+args[i])
		case name == "type":
			typeWithoutValue = true
		default:
			result = append(result, "--"+name+"=")
		}
	}
	switch {
	case len(unknown) == 1:
		return nil, "Unknown argument: " + unknown[0]
	case len(unknown) > 1:
		return nil, "Unknown arguments: " + strings.Join(unknown, ", ")
	case typeWithoutValue:
		return nil, invalidTypeMessage("true")
	}
	return result, ""
}

// validateArgs is the checks yargs runs after .strict(), in its order: .choices(), then .implies().
func validateArgs(flags *flag.FlagSet) string {
	isSet := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { isSet[f.Name] = true })
	if t := flags.Lookup("type").Value.String(); isSet["type"] && t != "game" && t != "model" && t != "package" {
		return invalidTypeMessage(`"` + t + `"`)
	}
	isTrue := func(name string) bool { return flags.Lookup(name).Value.String() == "true" }
	if isTrue("usePolling") && !isTrue("watch") && !isTrue("w") {
		return "Missing dependent arguments:\n usePolling -> watch"
	}
	return ""
}

func isBoolFlag(f *flag.Flag) bool {
	boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && boolFlag.IsBoolFlag()
}

func kebabToCamel(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if i > 0 && part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func invalidTypeMessage(given string) string {
	return "Invalid values:\n  Argument: type, Given: " + given + `, Choices: "game", "model", "package"`
}
