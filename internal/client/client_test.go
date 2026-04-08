package client

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNew_defaults(t *testing.T) {
	c := New("auth", "csrf", nil)
	if c.authToken != "auth" {
		t.Errorf("authToken: want %q, got %q", "auth", c.authToken)
	}
	if c.ct0 != "csrf" {
		t.Errorf("ct0: want %q, got %q", "csrf", c.ct0)
	}
	if c.clientUUID == "" {
		t.Error("clientUUID should be generated")
	}
	if c.deviceID == "" {
		t.Error("deviceID should be generated")
	}
	if c.httpClient == nil {
		t.Fatal("httpClient should not be nil")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("default timeout: want 30s, got %v", c.httpClient.Timeout)
	}
	if c.queryIDCache == nil {
		t.Fatal("queryIDCache should be initialized")
	}
}

func TestNew_uuidsAreUnique(t *testing.T) {
	c1 := New("a", "b", nil)
	c2 := New("a", "b", nil)
	if c1.clientUUID == c2.clientUUID {
		t.Error("two clients should have different clientUUIDs")
	}
	if c1.deviceID == c2.deviceID {
		t.Error("two clients should have different deviceIDs")
	}
	if c1.clientUUID == c1.deviceID {
		t.Error("clientUUID and deviceID should differ within same client")
	}
}

func TestNew_customHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c := New("a", "b", &Options{HTTPClient: custom})
	if c.httpClient != custom {
		t.Error("should use provided HTTPClient")
	}
}

func TestNew_customTimeout(t *testing.T) {
	c := New("a", "b", &Options{TimeoutMs: 100})
	if c.httpClient.Timeout != 100*time.Millisecond {
		t.Errorf("timeout: want 100ms, got %v", c.httpClient.Timeout)
	}
}

func TestNew_customTimeoutIgnoredWhenHTTPClientSet(t *testing.T) {
	custom := &http.Client{Timeout: 99 * time.Second}
	c := New("a", "b", &Options{HTTPClient: custom, TimeoutMs: 100})
	if c.httpClient.Timeout != 99*time.Second {
		t.Errorf("should use custom client timeout, got %v", c.httpClient.Timeout)
	}
}

func TestNew_queryIDCacheSeeded(t *testing.T) {
	seed := map[string]string{"Op1": "qid1", "Op2": "qid2"}
	c := New("a", "b", &Options{QueryIDCache: seed})
	if c.queryIDCache["Op1"] != "qid1" {
		t.Errorf("Op1: want qid1, got %q", c.queryIDCache["Op1"])
	}
	if c.queryIDCache["Op2"] != "qid2" {
		t.Errorf("Op2: want qid2, got %q", c.queryIDCache["Op2"])
	}
}

func TestNew_queryIDCacheIsCopied(t *testing.T) {
	seed := map[string]string{"Op": "qid"}
	c := New("a", "b", &Options{QueryIDCache: seed})
	seed["Op"] = "changed"
	if c.queryIDCache["Op"] != "qid" {
		t.Error("seed mutation should not affect client cache")
	}
}

func TestNew_nilOptions(t *testing.T) {
	c := New("a", "b", nil)
	if c.httpClient == nil {
		t.Fatal("httpClient should be created with defaults")
	}
}

func TestNew_emptyOptions(t *testing.T) {
	c := New("a", "b", &Options{})
	if c.httpClient == nil {
		t.Fatal("httpClient should be created with defaults for empty Options")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("default timeout with empty opts: want 30s, got %v", c.httpClient.Timeout)
	}
}

func TestNew_zeroTimeoutUsesDefault(t *testing.T) {
	c := New("a", "b", &Options{TimeoutMs: 0})
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("zero timeout should use default 30s, got %v", c.httpClient.Timeout)
	}
}

func TestNew_negativeTimeoutUsesDefault(t *testing.T) {
	c := New("a", "b", &Options{TimeoutMs: -100})
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("negative timeout should use default 30s, got %v", c.httpClient.Timeout)
	}
}

func TestCachedUserID_empty(t *testing.T) {
	c := New("a", "b", nil)
	if id := c.cachedUserID(); id != "" {
		t.Errorf("cachedUserID: want empty, got %q", id)
	}
}

func TestCachedUserID_afterSet(t *testing.T) {
	c := New("a", "b", nil)
	c.userID = "12345"
	if id := c.cachedUserID(); id != "12345" {
		t.Errorf("cachedUserID: want 12345, got %q", id)
	}
}

func TestNew_emptyQueryIDCache(t *testing.T) {
	c := New("a", "b", &Options{QueryIDCache: map[string]string{}})
	if len(c.queryIDCache) != 0 {
		t.Errorf("empty seed should result in empty cache, got %d entries", len(c.queryIDCache))
	}
}

func TestWaitForRateLimit_ConcurrentReservationsAreSerialized(t *testing.T) {
	c := New("a", "b", &Options{RequestsPerSecond: 40})
	start := make(chan struct{})
	times := make(chan time.Time, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- c.waitForRateLimit(context.Background())
			times <- time.Now()
		}()
	}

	close(start)
	wg.Wait()
	close(errs)
	close(times)

	for err := range errs {
		if err != nil {
			t.Fatalf("waitForRateLimit: unexpected error: %v", err)
		}
	}

	got := make([]time.Time, 0, 2)
	for ts := range times {
		got = append(got, ts)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 timestamps, got %d", len(got))
	}

	diff := got[0].Sub(got[1])
	if diff < 0 {
		diff = -diff
	}
	if diff < 20*time.Millisecond {
		t.Fatalf("concurrent reservations should be noticeably separated, got %v", diff)
	}
}

func TestWaitForRateLimit_HonorsContextCancellation(t *testing.T) {
	c := New("a", "b", &Options{RequestsPerSecond: 1})
	if err := c.waitForRateLimit(context.Background()); err != nil {
		t.Fatalf("first waitForRateLimit: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := c.waitForRateLimit(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForRateLimit with canceled context = %v, want %v", err, context.Canceled)
	}
}

func TestWaitForRateLimit_CanceledWaitDoesNotConsumeExtraSlot(t *testing.T) {
	c := New("a", "b", &Options{RequestsPerSecond: 40})
	if err := c.waitForRateLimit(context.Background()); err != nil {
		t.Fatalf("first waitForRateLimit: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.waitForRateLimit(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForRateLimit with canceled context = %v, want %v", err, context.Canceled)
	}

	start := time.Now()
	if err := c.waitForRateLimit(context.Background()); err != nil {
		t.Fatalf("third waitForRateLimit: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 60*time.Millisecond {
		t.Fatalf("canceled wait should not consume an extra slot, got elapsed %v", elapsed)
	}
}
