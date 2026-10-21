package core

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxTagsPerImage = 20
	maxTagRunes     = 32
	maxNameRunes    = 200
)

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// normalizeTag trims one tag and checks the tag rules.
func normalizeTag(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", validation("Tags must be valid UTF-8 text.")
	}
	t := strings.TrimSpace(raw)
	switch n := utf8Len(t); {
	case n == 0:
		return "", validation("A tag must not be empty.")
	case n > maxTagRunes:
		return "", validation("A tag must be at most %d characters.", maxTagRunes)
	}
	switch {
	case hasControl(t):
		return "", validation("A tag must not contain control characters.")
	case strings.Contains(t, ","):
		return "", validation("A tag must not contain commas.")
	case strings.HasPrefix(t, "#"):
		return "", validation("A tag must not start with #.")
	}
	return t, nil
}

// normalizeTags validates a tag list, removes exact duplicates and returns it sorted.
func normalizeTags(in []string) ([]string, error) {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		t, err := normalizeTag(raw)
		if err != nil {
			return nil, err
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	if len(out) > maxTagsPerImage {
		return nil, validation("An image can have at most %d tags.", maxTagsPerImage)
	}
	sort.Strings(out)
	return out, nil
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
