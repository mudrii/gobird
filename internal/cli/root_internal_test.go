package cli

import (
	"context"
	"testing"

	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/testutil"
)

func uploadMediaError(t *testing.T, status int) error {
	t.Helper()

	srv := testutil.NewTestServer(testutil.StaticHandler(status, `{"error":"boom"}`))
	t.Cleanup(srv.Close)

	c := client.New("fake-auth", "fake-ct0", &client.Options{
		HTTPClient: testutil.NewHTTPClientForServer(srv),
	})
	_, err := c.UploadMedia(context.Background(), []byte("x"), "image/png", "")
	if err == nil {
		t.Fatalf("UploadMedia() with HTTP %d: expected error", status)
	}
	return err
}

func TestExitCode_HTTPStatusMappings(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   int
	}{
		{name: "unauthorized", status: 401, want: 3},
		{name: "forbidden", status: 403, want: 3},
		{name: "rate_limit", status: 429, want: 4},
		{name: "server_error", status: 500, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := uploadMediaError(t, tc.status)
			if got := ExitCode(err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", err, got, tc.want)
			}
		})
	}
}
