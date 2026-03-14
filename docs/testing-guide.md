# Testing Guide

## Test Structure Overview

Tests are organized by package. The main test locations are:

| Location | Type | Notes |
|---|---|---|
| `internal/client/*_test.go` | Unit + mock HTTP | All API operation tests |
| `internal/parsing/*_test.go` | Unit | Pure function tests on wire JSON |
| `internal/auth/resolve_test.go` | Unit | Credential resolution logic |
| `internal/cli/root_test.go` | Unit | CLI flag and wiring tests |
| `internal/config/config_test.go` | Unit | Config loading |
| `internal/output/*_test.go` | Unit | Formatting |
| `tests/acceptance/` | CLI acceptance | End-to-end command tests |
| `tests/integration/` | Integration | Require real credentials, not run by default |

Browser cookie extraction code (`chrome.go`, `safari.go`, `firefox.go`) is not unit-tested — it requires a real browser profile and macOS Keychain access.

The scraper function `scrapeQueryIDs` is also not unit-tested — it requires network access to x.com. Tests inject a no-op scraper via `Client.scraper`.

---

## Using `testutil.NewTestServer()` for Mock HTTP

`internal/testutil/httpmock.go` provides the primitives for standing up an in-process HTTP server. The pattern used throughout `internal/client/` is:

```go
func TestMyOperation(t *testing.T) {
    // 1. Create an http.Handler that returns the wire response
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(200)
        _, _ = w.Write(myWireJSON)
    })

    // 2. Start the test server
    srv := testutil.NewTestServer(handler)
    defer srv.Close()

    // 3. Build a client wired to the test server
    c := client.New("tok", "ct0tok", &client.Options{
        HTTPClient: testutil.NewHTTPClientForServer(srv),
        QueryIDCache: map[string]string{
            "MyOperation": "testqid",
        },
    })
    // Prevent the real query ID scraper from making network calls
    // (only needed if the operation under test can trigger refreshQueryIDs)

    result, err := c.GetMyNewData(context.Background(), "123", nil)
    if err != nil {
        t.Fatal(err)
    }
    // assert on result
}
```

`testutil.NewHTTPClientForServer(srv)` returns an `*http.Client` whose transport rewrites every request's host to `srv.URL` while preserving path and query string. This means the client code does not need any special test mode — it makes its normal HTTP calls, which are silently redirected to the in-process server.

---

## Using `testutil.StaticHandler()` for Static Responses

For simple test cases that always return one fixed response:

```go
body := `{"data":{"create_tweet":{"tweet_results":{"result":{"rest_id":"99999"}}}}}`
srv := testutil.NewTestServer(testutil.StaticHandler(200, body))
defer srv.Close()
```

`StaticHandler(code, body)` always responds with the given HTTP status code, `Content-Type: application/json`, and the given body string. Use it when the handler does not need to inspect the request.

---

## Building Realistic Wire JSON Fixtures

Twitter's wire format is deeply nested. Build fixture JSON from the inside out, matching the actual wire shape. Key paths for common operations:

**Tweet from timeline instructions** (`data.search_by_raw_query.search_timeline.timeline.instructions`):
```json
{
  "data": {
    "search_by_raw_query": {
      "search_timeline": {
        "timeline": {
          "instructions": [
            {
              "type": "TimelineAddEntries",
              "entries": [
                {
                  "entryId": "tweet-123",
                  "sortIndex": "123",
                  "content": {
                    "entryType": "TimelineTimelineItem",
                    "itemContent": {
                      "__typename": "TimelineTweet",
                      "tweet_results": {
                        "result": {
                          "rest_id": "123",
                          "is_blue_verified": false,
                          "legacy": {
                            "full_text": "hello",
                            "created_at": "Mon Jan 01 00:00:00 +0000 2024",
                            "user_id_str": "456",
                            "reply_count": 0,
                            "retweet_count": 0,
                            "favorite_count": 0
                          },
                          "core": {
                            "user_results": {
                              "result": {
                                "rest_id": "456",
                                "legacy": {
                                  "screen_name": "testuser",
                                  "name": "Test User"
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              ]
            }
          ]
        }
      }
    }
  }
}
```

**Cursor entry** (placed as a sibling entry in the same instruction's `entries` list):
```json
{
  "entryId": "cursor-bottom-0",
  "sortIndex": "0",
  "content": {
    "entryType": "TimelineTimelineCursor",
    "cursorType": "Bottom",
    "value": "next_cursor_value_here"
  }
}
```

Note: `cursorType` is at `entry.content.cursorType`, not at `entry.content.itemContent.cursorType`. This is correction #7.

**Single tweet** (for `GetTweet`, path `data.tweetResult.result`):
```json
{
  "data": {
    "tweetResult": {
      "result": {
        "rest_id": "123",
        "is_blue_verified": true,
        "legacy": { "full_text": "hello" }
      }
    }
  }
}
```

Note: `tweetResult` is camelCase here (correction #58).

**Home timeline** (path `data.home.home_timeline_urt.instructions`):
```json
{
  "data": {
    "home": {
      "home_timeline_urt": {
        "instructions": [ ... ]
      }
    }
  }
}
```

This is correction #60 — the path uses snake_case `home_timeline_urt`, not camelCase.

---

## Table-Driven Test Patterns

The codebase uses standard Go table-driven tests. The naming convention is `TestFuncName_Scenario`:

```go
func TestGetTweet_Success(t *testing.T) { ... }
func TestGetTweet_NotFound(t *testing.T) { ... }
func TestGetTweet_404TriggersRefresh(t *testing.T) { ... }
```

For table-driven tests covering multiple wire shapes:

```go
func TestMapTweetResult(t *testing.T) {
    cases := []struct {
        name string
        raw  *types.WireRawTweet
        want *types.TweetData
    }{
        {
            name: "basic tweet",
            raw:  &types.WireRawTweet{RestID: "1", Legacy: &types.WireTweetLegacy{FullText: "hi"}},
            want: &types.TweetData{ID: "1", Text: "hi"},
        },
        {
            name: "visibility wrapper unwrapped",
            raw:  &types.WireRawTweet{Tweet: &types.WireRawTweet{RestID: "2"}},
            want: &types.TweetData{ID: "2"},
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := parsing.MapTweetResult(tc.raw)
            // compare got vs tc.want
        })
    }
}
```

---

## Race Detector Usage

Always run `make test-race` (or `go test -race ./...`) before committing. The `Client` struct has several concurrent access patterns that are guarded by mutexes:

- `queryIDMu` (`sync.RWMutex`) guards `queryIDCache` and `queryIDRefreshAt`
- `userIDMu` (`sync.RWMutex`) guards `userID`

The race detector will catch any direct field access that bypasses these locks. The deliberate pattern is:

- Read: `c.queryIDMu.RLock()` / `c.queryIDMu.RUnlock()`
- Write: `c.queryIDMu.Lock()` / `c.queryIDMu.Unlock()`
- For `userID` specifically: always use `c.cachedUserID()` for reads, never `c.userID` directly

The CI target `make ci` runs `vet`, `test`, `test-race`, and `build` in sequence.

---

## Golden File Testing with `AssertGolden`

Golden files store expected output for complex serialization tests. They live under `tests/golden/`.

```go
import "github.com/mudrii/gobird/internal/testutil"

func TestFormatTweet(t *testing.T) {
    tweet := types.TweetData{ID: "1", Text: "hello"}
    got := formatTweetToBytes(tweet)

    testutil.AssertGolden(t, "testdata/format_tweet.golden", got)
}
```

To create or regenerate golden files after intentional output changes:

```bash
go test ./... -update
```

The `-update` flag is registered as a package-level variable in `testutil/golden.go`. If the golden file does not exist and `-update` is not passed, the test fails with a message telling you to run with `-update`.

Do not commit a changed golden file without verifying the new output is correct.

---

## Testing Pagination (Multi-Page Mock Responses)

To test multi-page behavior, build a handler that tracks request count and returns different responses per page:

```go
page := 0
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    switch page {
    case 0:
        page++
        // Return page 1 response with a cursor
        _, _ = w.Write(buildPageWithCursor([]string{"tweet1", "tweet2"}, "cursor-page2"))
    case 1:
        page++
        // Return page 2 response with no cursor (terminates pagination)
        _, _ = w.Write(buildPageWithCursor([]string{"tweet3"}, ""))
    default:
        // Should not be called — test will catch this
        t.Errorf("unexpected third page request")
        w.WriteHeader(500)
    }
})
```

For `paginateInline`-based operations (Search, Home, Bookmarks), pagination stops when:
1. `nextCursor` is empty
2. `nextCursor == cursor` (no progress)
3. `page.tweets` is empty
4. `added == 0` (all items already seen)
5. `maxPages` reached
6. `len(accumulated) >= limit` (when limit > 0)

For `paginateCursor`-based operations (TweetDetail replies/thread), pagination stops on:
1. Empty cursor
2. Cursor unchanged from previous page

Note: zero items on a page does NOT stop `paginateCursor` — only the cursor state matters.

---

## Testing Error Paths: 404 and Refresh Flow

The 404-then-refresh flow is testable by injecting a scraper that returns known query IDs and a handler that returns 404 on the first attempt, then 200 on the second:

```go
attempts := 0
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    attempts++
    if attempts == 1 {
        // First attempt: stale query ID → 404
        http.Error(w, `{"errors":[{"code":34}]}`, 404)
        return
    }
    // Second attempt (after refresh): success
    w.Header().Set("Content-Type", "application/json")
    _, _ = w.Write(successResponse)
})

srv := testutil.NewTestServer(handler)
defer srv.Close()

c := client.New("tok", "ct0", &client.Options{
    HTTPClient: testutil.NewHTTPClientForServer(srv),
    QueryIDCache: map[string]string{"MyOperation": "old-id"},
})
// Inject a scraper that produces a "new" query ID after refresh
c.SetScraper(func(_ context.Context) map[string]string {
    return map[string]string{"MyOperation": "new-id"}
})
```

For operations that use `withRefreshedQueryIDsOn404` (like `getFollowing`, `getFollowers`, `getUserAboutAccount`), the same pattern applies.

For testing `RefreshQueryIDs` itself: pass a pre-cancelled context to ensure the scraper's HTTP call is immediately cancelled without making real network requests:

```go
ctx, cancel := context.WithCancel(context.Background())
cancel() // cancel immediately
c.RefreshQueryIDs(ctx) // safe to call — scraper sees cancelled context
```

---

## Coverage Targets and How to Measure

```bash
make coverage
# Produces coverage.out and coverage.html

# View package-level coverage summary
go tool cover -func=coverage.out

# View specific package
go test -coverprofile=cov.out ./internal/client/...
go tool cover -html=cov.out
```

There are no enforced numeric coverage targets. Focus coverage on:
- All response parsing paths (happy path + nil-safety)
- All pagination stop conditions
- Error wrapping (ensure errors propagate with context)
- Query ID fallback chain

Do not target 100% coverage on:
- `scrapeQueryIDs` (requires network)
- `extractChrome`, `extractSafari`, `extractFirefox` (require OS state)
- `cmd/gobird/main.go` (requires process-level testing)

---

## What NOT to Test

The following are explicitly excluded from unit tests due to external dependencies:

- **`scrapeQueryIDs`**: Makes real HTTP requests to x.com. Tests inject a no-op `c.scraper` function (`func(_ context.Context) map[string]string { return nil }`) to prevent network calls.
- **Browser cookie extraction** has both unit and manual coverage:
  - unit tests cover Chrome decryption variants, Safari binarycookies parsing, and Firefox profile scanning using temp fixtures
  - real browser/keychain behavior is still validated manually or via environment-specific acceptance runs
- **Integration tests** (`tests/integration/`): Require real `AUTH_TOKEN` and `CT0` environment variables. Run them explicitly, not via `make test`.

---

## Test Naming Conventions

All test functions follow `TestFuncName_Scenario`:

```
TestGetTweet_Success
TestGetTweet_404RefreshesAndSucceeds
TestGetTweet_HTTPError
TestGetTweet_EmptyResult
TestExtractCursorFromInstructions_BottomCursor
TestExtractCursorFromInstructions_NoCursor
TestMapTweetResult_VisibilityWrapperUnwrapped
TestMapTweetResult_NilRaw
TestFollow_RESTSuccess
TestFollow_RESTBlockedReturnsError
TestFollow_FallsBackToGraphQL
```

Subtest names (in `t.Run(...)`) use natural language:

```go
t.Run("returns error when ct0 is missing", func(t *testing.T) { ... })
```

Test helper functions that call `t.Helper()` are lower-case and unexported.
