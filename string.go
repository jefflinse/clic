package handyman

import "strings"

func toUnderscores(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}

func toDashes(str string) string {
	return strings.ReplaceAll(str, "_", "-")
}
