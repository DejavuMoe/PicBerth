package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxNameRunes   = 200
	maxFolderRunes = 64
)

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// normalizeFilename validates a user-chosen display name.
func normalizeFilename(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", validation("filename must be valid UTF-8 text.")
	}
	n := strings.TrimSpace(raw)
	switch c := utf8Len(n); {
	case c == 0:
		return "", validation("filename must not be empty.")
	case c > maxNameRunes:
		return "", validation("filename must be at most %d characters.", maxNameRunes)
	}
	if hasControl(n) {
		return "", validation("filename must not contain control characters.")
	}
	if strings.ContainsAny(n, `/\`) {
		return "", validation("filename must not contain / or \\.")
	}
	return n, nil
}
