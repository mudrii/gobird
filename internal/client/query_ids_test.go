package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
)

func TestGetQueryID_RuntimeCacheFirst(t *testing.T) {
	c := New("tok", "ct0", &Options{
		QueryIDCache: map[string]string{
			"TweetDetail": "cached-id-xyz",
		},
	})
	id := c.getQueryID("TweetDetail")
	if id != "cached-id-xyz" {
		t.Errorf("getQueryID: want cached-id-xyz, got %q", id)
	}
}

func TestGetQueryID_FallsBackToBundled(t *testing.T) {
	// No cache entry for TweetDetail; should use BundledBaselineQueryIDs.
	c := New("tok", "ct0", nil)
	bundled := BundledBaselineQueryIDs["TweetDetail"]
	if bundled == "" {
		t.Skip("TweetDetail not in BundledBaselineQueryIDs")
	}
	id := c.getQueryID("TweetDetail")
	if id != bundled {
		t.Errorf("getQueryID: want bundled %q, got %q", bundled, id)
	}
}

func TestGetQueryID_FallsBackToFallback(t *testing.T) {
	// Operation not in bundled — should fall to FallbackQueryIDs.
	// "UserArticlesTweets" is in FallbackQueryIDs but not BundledBaselineQueryIDs.
	op := "UserArticlesTweets"
	if _, inBundled := BundledBaselineQueryIDs[op]; inBundled {
		t.Skipf("%s is in BundledBaselineQueryIDs, can't test fallback path", op)
	}
	fallback := FallbackQueryIDs[op]
	if fallback == "" {
		t.Skipf("%s not in FallbackQueryIDs either", op)
	}
	c := New("tok", "ct0", nil)
	id := c.getQueryID(op)
	if id != fallback {
		t.Errorf("getQueryID: want fallback %q, got %q", fallback, id)
	}
}

func TestGetQueryIDs_Deduplication(t *testing.T) {
	// When the runtime primary ID equals an entry in PerOperationFallbackIDs,
	// the result should not contain duplicates.
	op := "TweetDetail"
	primary := PerOperationFallbackIDs[op][0] // first fallback is the bundled one
	c := New("tok", "ct0", &Options{
		QueryIDCache: map[string]string{
			op: primary,
		},
	})
	ids := c.getQueryIDs(op)
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("getQueryIDs: duplicate ID %q in result", id)
		}
		seen[id] = true
	}
	if ids[0] != primary {
		t.Errorf("getQueryIDs: want primary %q first, got %q", primary, ids[0])
	}
}

func TestWithRefreshedQueryIDsOn404_NoRefreshOnSuccess(t *testing.T) {
	c := New("tok", "ct0", nil)
	c.scraper = func(_ context.Context) map[string]string { return nil }

	refreshCalled := false
	origScraper := c.scraper
	c.scraper = func(ctx context.Context) map[string]string {
		refreshCalled = true
		return origScraper(ctx)
	}

	ar, refreshed := c.withRefreshedQueryIDsOn404(context.Background(), func() attemptResult {
		return attemptResult{body: []byte(`{}`), success: true}
	})
	if refreshed {
		t.Error("withRefreshedQueryIDsOn404: should not refresh on success")
	}
	if refreshCalled {
		t.Error("withRefreshedQueryIDsOn404: scraper should not be called on success")
	}
	if !ar.success {
		t.Error("withRefreshedQueryIDsOn404: expected success=true")
	}
}

func TestWithRefreshedQueryIDsOn404_RefreshesOn404(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"ok":true}`))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"TweetDetail": "some-query-id",
	})

	scraperCalls := 0
	c.scraper = func(_ context.Context) map[string]string {
		scraperCalls++
		return map[string]string{"TweetDetail": "refreshed-id"}
	}

	callCount := 0
	ar, refreshed := c.withRefreshedQueryIDsOn404(context.Background(), func() attemptResult {
		callCount++
		if callCount == 1 {
			return attemptResult{err: &httpError{StatusCode: 404, Body: "not found"}, had404: true}
		}
		return attemptResult{body: []byte(`{}`), success: true}
	})

	if !refreshed {
		t.Error("withRefreshedQueryIDsOn404: should report refresh=true after 404")
	}
	if scraperCalls == 0 {
		t.Error("withRefreshedQueryIDsOn404: scraper not called after 404")
	}
	if callCount != 2 {
		t.Errorf("withRefreshedQueryIDsOn404: want 2 attempt calls, got %d", callCount)
	}
	if !ar.success {
		t.Errorf("withRefreshedQueryIDsOn404: expected success=true on retry, got err=%v", ar.err)
	}
}

func TestIs404(t *testing.T) {
	t.Run("404 error", func(t *testing.T) {
		err := &httpError{StatusCode: 404, Body: "not found"}
		if !is404(err) {
			t.Error("is404: expected true for 404 httpError")
		}
	})
	t.Run("500 error", func(t *testing.T) {
		err := &httpError{StatusCode: 500, Body: "server error"}
		if is404(err) {
			t.Error("is404: expected false for 500 httpError")
		}
	})
	t.Run("nil error", func(t *testing.T) {
		if is404(nil) {
			t.Error("is404: expected false for nil error")
		}
	})
}

func TestActiveQueryID_DelegatesToGetQueryID(t *testing.T) {
	c := New("tok", "ct0", &Options{
		QueryIDCache: map[string]string{"CreateTweet": "my-id"},
	})
	if got := c.ActiveQueryID("CreateTweet"); got != "my-id" {
		t.Errorf("ActiveQueryID: want my-id, got %q", got)
	}
}

func TestGetFollowing_404TriggersRefresh_NoRealHTTP(t *testing.T) {
	calls := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"Following": "mWYeougg_ocJS2Vr1Vt28w",
	})
	// scraper is already a no-op from newTestClientWith.
	_, _ = c.GetFollowing(context.Background(), "user-x", nil)
	if calls < 2 {
		t.Errorf("want at least 2 calls for 404+refresh pattern, got %d", calls)
	}
}

func TestScrapeBody_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`service unavailable`))
	}))
	defer srv.Close()

	_, err := scrapeBody(context.Background(), &http.Client{}, srv.URL)
	if err == nil {
		t.Fatal("expected non-2xx scrapeBody error")
	}
	status, ok := HTTPStatusCode(err)
	if !ok || status != 503 {
		t.Fatalf("want HTTP 503 error, got %v", err)
	}
}

func TestQueryIDFormatRe_Boundaries(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{input: "aaaaaaaaaaaaaaaaaaa", valid: false},  // 19 chars — too short
		{input: "aaaaaaaaaaaaaaaaaaaa", valid: true},   // 20 chars — minimum valid
		{input: "abcdefghijklmnopqrstuvwxyzABCDEF012345678901234567890", valid: true}, // 50 mixed valid
		{input: "aaaaaaaaaaaaaaaaaaaa!", valid: false}, // contains "!"
		{input: "", valid: false},                      // empty
		{input: "aaaaaaaaaaaaaaaaaaa a", valid: false}, // contains space
	}
	for _, tc := range cases {
		got := queryIDFormatRe.MatchString(tc.input)
		if got != tc.valid {
			t.Errorf("queryIDFormatRe.MatchString(%q) = %v, want %v", tc.input, got, tc.valid)
		}
	}
}

func TestRefreshQueryIDs_RejectsMalformedScrapedIDs(t *testing.T) {
	badIDs := map[string]string{
		"TweetDetail":  "tooshort",           // under 20 chars
		"CreateTweet":  "has a space in it!!!", // contains invalid chars
		"UserTweets":   "",                    // empty
	}
	c := New("tok", "ct0", nil)
	c.scraper = func(_ context.Context) map[string]string { return badIDs }

	c.refreshQueryIDs(context.Background())

	c.queryIDMu.RLock()
	defer c.queryIDMu.RUnlock()
	for op, bad := range badIDs {
		if got, ok := c.queryIDCache[op]; ok && got == bad && bad != "" {
			t.Errorf("queryIDCache[%q] = %q; malformed ID should have been rejected", op, got)
		}
	}
}

func TestRefreshQueryIDs_AcceptsValidScrapedIDs(t *testing.T) {
	const validID = "ValidQueryID123456789"
	c := New("tok", "ct0", nil)
	c.scraper = func(_ context.Context) map[string]string {
		return map[string]string{"TweetDetail": validID}
	}

	c.refreshQueryIDs(context.Background())

	c.queryIDMu.RLock()
	got := c.queryIDCache["TweetDetail"]
	c.queryIDMu.RUnlock()
	if got != validID {
		t.Errorf("queryIDCache[TweetDetail] = %q, want %q", got, validID)
	}
}
