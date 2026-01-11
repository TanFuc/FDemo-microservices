package usecase

import (
	"regexp"
	"strings"
)

var (
	slugRegex    = regexp.MustCompile(`[^a-z0-9]+`)
	multiDash    = regexp.MustCompile(`-+`)
)

// GenerateSlug creates a URL-friendly slug from the given text
func GenerateSlug(text string) string {
	slug := strings.ToLower(text)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = multiDash.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
