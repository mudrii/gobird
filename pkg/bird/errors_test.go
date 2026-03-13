package bird_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mudrii/gobird/pkg/bird"
)

func TestAPIError_Error(t *testing.T) {
	err := &bird.APIError{
		StatusCode: 429,
		Message:    "too many requests",
	}

	msg := err.Error()
	if !strings.Contains(msg, "429") {
		t.Fatalf("Error() should include status code, got %q", msg)
	}
	if !strings.Contains(msg, "too many requests") {
		t.Fatalf("Error() should include message, got %q", msg)
	}
}

func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    *bird.APIError
		target error
		want   bool
	}{
		{
			name:   "rate_limit",
			err:    &bird.APIError{StatusCode: 429, Message: "rate"},
			target: bird.ErrRateLimit,
			want:   true,
		},
		{
			name:   "unauthorized_401",
			err:    &bird.APIError{StatusCode: 401, Message: "unauthorized"},
			target: bird.ErrUnauthorized,
			want:   true,
		},
		{
			name:   "unauthorized_403",
			err:    &bird.APIError{StatusCode: 403, Message: "forbidden"},
			target: bird.ErrUnauthorized,
			want:   true,
		},
		{
			name:   "not_found",
			err:    &bird.APIError{StatusCode: 404, Message: "missing"},
			target: bird.ErrNotFound,
			want:   true,
		},
		{
			name:   "non_match",
			err:    &bird.APIError{StatusCode: 500, Message: "boom"},
			target: bird.ErrRateLimit,
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := errors.Is(tc.err, tc.target); got != tc.want {
				t.Fatalf("errors.Is(%v, %v) = %v, want %v", tc.err, tc.target, got, tc.want)
			}
		})
	}
}
