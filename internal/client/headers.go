package client

import "net/http"

// baseHeaders returns the standard request headers shared across all requests.
// These are the exact header names and values from the reference source.
func baseHeaders(authToken, ct0 string) http.Header {
	h := http.Header{}
	h.Set("authorization", "Bearer "+BearerToken)
	h.Set("x-twitter-auth-type", "OAuth2Session")
	h.Set("x-twitter-active-user", "yes")
	h.Set("x-twitter-client-language", "en")
	h.Set("x-csrf-token", ct0)
	h.Set("cookie", "auth_token="+authToken+"; ct0="+ct0)
	h.Set("user-agent", UserAgent)
	h.Set("origin", "https://x.com")
	h.Set("referer", "https://x.com/")
	return h
}

// jsonHeaders returns base headers plus content-type: application/json.
// Correction #70: getHeaders() delegates to getJsonHeaders() (includes content-type).
func jsonHeaders(authToken, ct0 string) http.Header {
	h := baseHeaders(authToken, ct0)
	h.Set("content-type", "application/json")
	return h
}

// uploadHeaders returns only the base headers for media upload requests.
// Correction #70: getUploadHeaders() = getBaseHeaders() only — no content-type override.
func uploadHeaders(authToken, ct0 string) http.Header {
	return baseHeaders(authToken, ct0)
}
