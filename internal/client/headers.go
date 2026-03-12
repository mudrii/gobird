package client

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

func createTransactionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

// baseHeaders returns the standard request headers shared across all requests.
// These are the exact header names and values from the reference source.
func baseHeaders(authToken, ct0, clientUUID, deviceID, userID string) http.Header {
	h := http.Header{}
	h.Set("accept", "*/*")
	h.Set("accept-language", "en-US,en;q=0.9")
	h.Set("authorization", "Bearer "+BearerToken)
	h.Set("x-twitter-auth-type", "OAuth2Session")
	h.Set("x-twitter-active-user", "yes")
	h.Set("x-twitter-client-language", "en")
	h.Set("x-csrf-token", ct0)
	if clientUUID != "" {
		h.Set("x-client-uuid", clientUUID)
	}
	if deviceID != "" {
		h.Set("x-twitter-client-deviceid", deviceID)
	}
	if txID := createTransactionID(); txID != "" {
		h.Set("x-client-transaction-id", txID)
	}
	if userID != "" {
		h.Set("x-twitter-client-user-id", userID)
	}
	h.Set("cookie", "auth_token="+authToken+"; ct0="+ct0)
	h.Set("user-agent", UserAgent)
	h.Set("origin", "https://x.com")
	h.Set("referer", "https://x.com/")
	return h
}

// jsonHeaders returns base headers plus content-type: application/json.
// Correction #70: getHeaders() delegates to getJsonHeaders() (includes content-type).
func jsonHeaders(authToken, ct0, clientUUID, deviceID, userID string) http.Header {
	h := baseHeaders(authToken, ct0, clientUUID, deviceID, userID)
	h.Set("content-type", "application/json")
	return h
}

// uploadHeaders returns only the base headers for media upload requests.
// Correction #70: getUploadHeaders() = getBaseHeaders() only — no content-type override.
func uploadHeaders(authToken, ct0, clientUUID, deviceID, userID string) http.Header {
	return baseHeaders(authToken, ct0, clientUUID, deviceID, userID)
}
