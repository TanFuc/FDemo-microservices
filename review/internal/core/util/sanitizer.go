package util

import (
	"net/url"
	"regexp"
	"strings"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// StripHTMLTags removes all HTML tags from the input string to prevent XSS
func StripHTMLTags(input string) string {
	// Remove HTML tags
	result := htmlTagRegex.ReplaceAllString(input, "")
	// Trim whitespace
	result = strings.TrimSpace(result)
	return result
}

// ValidateImageURL checks if the URL is valid and uses http or https
func ValidateImageURL(imageURL string) bool {
	if imageURL == "" {
		return false
	}

	parsed, err := url.Parse(imageURL)
	if err != nil {
		return false
	}

	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// ValidateImageURLs validates a slice of image URLs
func ValidateImageURLs(urls []string) []string {
	var valid []string
	for _, u := range urls {
		if ValidateImageURL(u) {
			valid = append(valid, u)
		}
	}
	return valid
}
