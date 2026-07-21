package store

import "strings"

func toLower(s string) string {
	return strings.ToLower(s)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func hasTag(tags []string, query string) bool {
	for _, t := range tags {
		if strings.Contains(toLower(t), query) {
			return true
		}
	}
	return false
}
