package bird

import (
	"errors"
	"fmt"
)

// ErrMissingCredentials is returned when auth_token or ct0 is empty.
var ErrMissingCredentials = errors.New("bird: auth_token and ct0 are required")

// errMissingCredentials is kept as an alias for internal use.
var errMissingCredentials = ErrMissingCredentials

// ErrRateLimit is returned when the API returns HTTP 429.
var ErrRateLimit = errors.New("bird: rate limit exceeded")

// ErrUnauthorized is returned when the API returns HTTP 401 or 403.
var ErrUnauthorized = errors.New("bird: unauthorized or forbidden")

// ErrNotFound is returned when the requested resource is not found.
var ErrNotFound = errors.New("bird: not found")

// APIError represents an error response from the Twitter/X API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bird: API error %d: %s", e.StatusCode, e.Message)
}

// Is supports errors.Is matching against sentinel errors.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrRateLimit:
		return e.StatusCode == 429
	case ErrUnauthorized:
		return e.StatusCode == 401 || e.StatusCode == 403
	case ErrNotFound:
		return e.StatusCode == 404
	}
	return false
}
