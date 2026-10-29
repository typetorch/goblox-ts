package transform

// Port of TSTransformer/util/cleanModuleName.ts.

import "strings"

// cleanModuleName is TSTransformer/util/cleanModuleName.ts cleanModuleName: `name.replace(/\W/g, "_")`.
// JavaScript replaces every UTF-16 code unit, so a rune outside the BMP becomes two underscores.
func cleanModuleName(name string) string {
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			sb.WriteRune(r)
		case r > 0xFFFF:
			sb.WriteString("__")
		default:
			sb.WriteByte('_')
		}
	}
	return sb.String()
}
