package auth

import (
	"context"

	"github.com/mudrii/gobird/internal/types"
)

// ExportedExtractWithTimeout is the exported wrapper of extractWithTimeout for testing.
func ExportedExtractWithTimeout(timeoutMs int, fn func(context.Context) (*types.TwitterCookies, error)) (*types.TwitterCookies, error) {
	return extractWithTimeout(timeoutMs, fn)
}

// ExportedNormalizeCookieSources is the exported wrapper of normalizeCookieSources for testing.
func ExportedNormalizeCookieSources(sources []string) ([]string, error) {
	return normalizeCookieSources(sources)
}

// ExportedExtractFromBrowserOrder is the exported wrapper of extractFromBrowserOrder for testing.
func ExportedExtractFromBrowserOrder(order []string, opts ResolveOptions) (*types.TwitterCookies, error) {
	return extractFromBrowserOrder(order, opts)
}
