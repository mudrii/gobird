# GoBird Implementation Prompt

## Objective

Build a Go implementation of `bird` that replicates the behavior of the reference project in:

- Source repo: `/Users/mudrii/src/bird`
- Documentation: `/Users/mudrii/src/bird-docs`
- Target repo for all development: `/Users/mudrii/src/gobird`

This is not a greenfield Twitter CLI. It is a source-compatible rewrite of the existing `bird` behavior into idiomatic Go.

If the docs and naive assumptions conflict, follow the docs.
If the docs and compiled source conflict, follow the compiled source.
If this prompt conflicts with either, follow the source of truth and update the prompt or code accordingly.

## Source Of Truth

Read in this order before implementing anything:

1. `/Users/mudrii/src/bird-docs/corrections.md`
2. `/Users/mudrii/src/bird-docs/README.md`
3. `/Users/mudrii/src/bird-docs/cli-commands.md`
4. `/Users/mudrii/src/bird-docs/graphql-operations.md`
5. `/Users/mudrii/src/bird-docs/api-endpoints.md`
6. `/Users/mudrii/src/bird-docs/features-flags.md`
7. `/Users/mudrii/src/bird-docs/response-parsing.md`
8. `/Users/mudrii/src/bird-docs/data-types.md`
9. `/Users/mudrii/src/bird-docs/authentication.md`
10. `/Users/mudrii/src/bird-docs/architecture.md`
11. `/Users/mudrii/src/bird-docs/go-implementation-guide.md`

Do not invent commands, flags, payloads, paths, feature maps, or response shapes.

## Non-Negotiable Requirements

- Language: Go
- Go version: 1.23+
- Go module path: `github.com/mudrii/gobird`
- Development repo: `/Users/mudrii/src/gobird`
- Use TDD and ATDD
- Strong typing throughout
- All public APIs and data models must be statically typed
- All exported identifiers must have doc comments
- Use `context.Context` as the first parameter for all networked operations
- Lint with `golangci-lint`
- Run `go test`, `go test -race`, `go vet`, and lint in CI
- Keep code idiomatic Go, but preserve `bird` behavior exactly where behavior is observable
- Never log secrets such as `auth_token` or `ct0`
- Never ship dead code, commented-out code, or placeholder stubs

## Primary Goal

Replicate all functional behavior of the reference `bird` implementation:

- CLI behavior
- Output behavior
- JSON output contracts
- Request payloads
- Feature flags
- Query ID refresh logic
- Authentication flow
- Pagination semantics
- Error handling and fallbacks
- Response normalization

Exact behavioral parity matters more than architectural elegance.

## Scope Clarification

There are two separate parity targets:

1. CLI parity
2. Library/client parity

CLI parity must match the real `bird` CLI exactly.
Do not add new top-level CLI commands unless they exist in the reference CLI.

Library/client parity should cover the full underlying feature set used by the source project, including methods that are not directly exposed as standalone CLI commands.

## Project Layout

Use a Go layout close to this:

```text
gobird/
├── cmd/
│   └── bird/
│       └── main.go
├── internal/
│   ├── auth/
│   ├── client/
│   ├── cli/
│   ├── config/
│   ├── output/
│   ├── runtime/
│   └── types/
├── pkg/
│   ├── extract/
│   └── normalize/
├── tests/
│   └── fixtures/
├── .golangci.yml
├── Makefile
├── go.mod
└── README.md
```

Unit tests live alongside source files as `_test.go` files (idiomatic Go). Integration tests live in `tests/integration/` as a separate package. Fixtures live in `tests/fixtures/`. Do not create a `tests/unit/` directory.

Package breakdown should reflect the domains in the reference implementation:

- auth and cookie resolution
- HTTP client and headers
- query ID runtime cache
- feature flags and field toggles
- tweet parsing and normalization
- pagination
- bookmarks and thread filters
- media upload
- CLI command wiring
- output formatting

## Required CLI Surface

Implement the actual CLI surface from the reference project.

### Commands

- `bird tweet "<text>"`
- `bird reply <tweet-id-or-url> "<text>"`
- `bird read <tweet-id-or-url>`
- `bird <tweet-id-or-url>` as shorthand for `read`
- `bird replies <tweet-id-or-url>`
- `bird thread <tweet-id-or-url>`
- `bird search "<query>"`
- `bird mentions`
- `bird home`
- `bird bookmarks`
- `bird unbookmark <tweet-id-or-url...>`
- `bird likes`
- `bird news`
- `bird trending` as alias for `news`
- `bird user-tweets <@handle>`
- `bird lists`
- `bird list-timeline <list-id-or-url>`
- `bird following`
- `bird followers`
- `bird about <@handle>`
- `bird whoami`
- `bird check`
- `bird query-ids`
- `bird help [command]`

### Global Options

Support the real global flags:

- `--auth-token <token>`
- `--ct0 <token>`
- `--cookie-source <src>` repeatable, values: `safari`, `chrome`, `firefox`
- `--chrome-profile <name>`
- `--chrome-profile-dir <path>`
- `--firefox-profile <name>`
- `--cookie-timeout <ms>`
- `--timeout <ms>`
- `--quote-depth <n>`
- `--plain`
- `--no-emoji`
- `--no-color`
- `--media <path>` repeatable
- `--alt <text>` repeatable
- `--version`
- `--help`

Do not claim support for fake global flags such as `--dry-run` unless you add them intentionally and clearly document that they are an extension rather than parity behavior.

### Per-Command Flags

Honor the reference command flags and semantics, including:

- `--json`
- `--json-full` on `home`, `likes`, `news` — includes `_raw` field in output
- `-n <count>` / `--count <n>` on `search`, `mentions`, `home`, `bookmarks`, `likes`, `user-tweets`, `lists`, `list-timeline`, `following`, `followers`, `news`
- pagination flags `--all`, `--max-pages`, `--cursor` on `replies`, `thread`, `search`, `bookmarks`, `list-timeline`, `following`, `followers`, `user-tweets`
- `--delay <ms>` on `replies`, `thread`, `user-tweets` (default 1000ms)
- `--following` on `home`
- `--folder-id` on `bookmarks`
- bookmark thread expansion flags:
  - `--expand-root-only`
  - `--author-chain`
  - `--author-only`
  - `--full-chain-only`
  - `--include-ancestor-branches`
  - `--include-parent`
  - `--thread-meta`
  - `--sort-chronological`
- `--member-of` on `lists`
- `--user <userId>` on `following` and `followers` (takes a numeric user ID, not a @handle)
- `--user <@handle>` on `mentions` (takes a @handle, not a user ID)
- `--fresh` on `query-ids`
- news tab flags:
  - `--ai-only`
  - `--with-tweets`
  - `--tweets-per-item`
  - `--for-you`
  - `--news-only`
  - `--sports`
  - `--entertainment`
  - `--trending-only`

Use `/Users/mudrii/src/bird-docs/cli-commands.md` as the CLI contract.

### Exit Codes

- `0`: success
- `1`: runtime error (network, API, auth, partial pagination failure)
- `2`: invalid usage or validation error (bad handle, missing required arg)

Partial pagination results (some items fetched before error) exit with code `1` but still print accumulated items to stdout before the error message.

## Required Library/Client Surface

Implement a Go client that can support the behavior of the reference implementation, including:

- tweet
- reply
- get tweet
- get replies
- get thread
- search
- mentions flow
- home timeline
- latest/following timeline
- bookmarks
- bookmark folder timeline
- unbookmark
- likes
- like
- unlike
- retweet
- unretweet
- bookmark mutation
- follow
- unfollow
- get user tweets
- get current user
- get following
- get followers
- get user ID by username
- get user about account
- get owned lists
- get list memberships
- get list timeline
- get news
- upload media

Library parity may exceed CLI parity. That is acceptable.

## Core Data Contracts

Model the real data contracts from `/Users/mudrii/src/bird-docs/data-types.md`.

At minimum support:

- `TweetData`
- `TweetWithMeta`
- `TweetAuthor`
- `TweetMedia`
- `TweetArticle`
- `GetTweetResult`
- `SearchResult`
- `TweetResult`
- `MutationResult`
- `FollowMutationResult`
- `TwitterUser`
- `FollowingResult`
- `CurrentUserResult`
- `UserLookupResult`
- `TwitterList`
- `ListsResult`
- `NewsItem`
- `NewsResult`
- `AboutAccountProfile`
- `AboutAccountResult`
- `UploadMediaResult`

Important output contract details:

- `TweetData` supports `_raw` when full JSON output is requested
- bookmark `--thread-meta` output requires `TweetWithMeta`
- article output shape must match the documented type
- JSON output must match the reference CLI format closely

## Authentication

Use cookie-based auth only.

Required cookies:

- `auth_token`
- `ct0`

Resolution order:

1. CLI flags
2. Environment variables
3. Browser cookies in configured order

Environment variables:

- `AUTH_TOKEN` or `TWITTER_AUTH_TOKEN`
- `CT0` or `TWITTER_CT0`
- `BIRD_TIMEOUT_MS`
- `BIRD_COOKIE_TIMEOUT_MS`
- `BIRD_QUOTE_DEPTH`
- `BIRD_QUERY_IDS_CACHE`
- `NO_COLOR` — disables ANSI color output (same effect as `--no-color`)
- `BIRD_DEBUG_BOOKMARKS` — set to `1` to enable verbose bookmark retry logging (non-parity extension)

Config file locations:

- `~/.config/bird/config.json5`
- `./.birdrc.json5`

Config file permissions must be `0600`.

## Browser Cookie Extraction

Implement cookie extraction for:

- Safari
- Chrome
- Firefox

Reference paths and behavior are documented in `/Users/mudrii/src/bird-docs/authentication.md`.

Support:

- explicit browser order
- explicit Chrome profile
- explicit Chrome profile dir / cookie DB path
- explicit Firefox profile
- extraction timeout

**Platform complexity warning**: This is the highest-complexity platform-specific piece of the implementation.

- **Chrome**: Cookies are stored in an SQLite database at `~/Library/Application Support/Google/Chrome/<Profile>/Cookies`. Cookie values are encrypted with AES-256-CBC using a key stored in the macOS Keychain under `Chrome Safe Storage`. You must retrieve the key via the macOS Security framework (or `security` CLI) and decrypt each value before use.
- **Safari**: Cookies are stored in a proprietary binary format at `~/Library/Containers/com.apple.Safari/Data/Library/Cookies/Cookies.binarycookies` on modern macOS. This format requires a custom parser (not SQLite).
- **Firefox**: Cookies are stored in plain SQLite at `~/Library/Application Support/Firefox/Profiles/<profile>/cookies.sqlite`. No decryption required.

Read `authentication.md` in full before implementing any browser cookie extractor.

## HTTP Headers

Every request must use the documented base headers, including:

- `accept`
- `accept-language`
- `authorization: Bearer <web app token>`
- `x-csrf-token`
- `x-twitter-auth-type`
- `x-twitter-active-user`
- `x-twitter-client-language`
- `x-client-uuid`
- `x-twitter-client-deviceid`
- `x-client-transaction-id`
- `cookie`
- `user-agent`
- `origin`
- `referer`

Additional rules:

- use `content-type: application/json` for JSON POST
- do not manually set multipart content type for media upload
- for tweet/reply and `statuses/update.json` fallback, use `referer: https://x.com/compose/post`
- once current user is known, also send `x-twitter-client-user-id`

## GraphQL And REST Operations

Base GraphQL URL:

`https://x.com/i/api/graphql`

Implement the full baseline query ID map from `/Users/mudrii/src/bird-docs/graphql-operations.md`, not just a subset.

This includes, at minimum:

- `CreateTweet`
- `CreateRetweet`
- `DeleteRetweet`
- `CreateFriendship`
- `DestroyFriendship`
- `FavoriteTweet`
- `UnfavoriteTweet`
- `CreateBookmark`
- `DeleteBookmark`
- `TweetDetail`
- `SearchTimeline`
- `UserArticlesTweets`
- `UserTweets`
- `Bookmarks`
- `Following`
- `Followers`
- `Likes`
- `BookmarkFolderTimeline`
- `ListOwnerships`
- `ListMemberships`
- `ListLatestTweetsTimeline`
- `ListByRestId`
- `HomeTimeline`
- `HomeLatestTimeline`
- `ExploreSidebar`
- `ExplorePage`
- `GenericTimelineById`
- `TrendHistory`
- `AboutAccountQuery`

Also implement special fallback/rotation behavior:

- TweetDetail fallback IDs
- SearchTimeline fallback IDs
- `UserByScreenName` hardcoded query ID rotation

## Critical Query ID Rules

- Maintain a runtime cache on disk
- Default cache file: `~/.config/bird/query-ids-cache.json`
- TTL: 24 hours
- Refresh on GraphQL 404
- Refresh on `GRAPHQL_VALIDATION_FAILED`
- Refresh when HomeTimeline/HomeLatestTimeline returns `"query: unspecified"`
- scrape IDs from X web bundles
- persist discovered bundle metadata

Important:

- `UserByScreenName` must not use the normal runtime cache
- it must rotate through the documented hardcoded IDs
- if response type is `UserUnavailable`, stop rotating and return error

## Feature Flags And Field Toggles

Treat `/Users/mudrii/src/bird-docs/features-flags.md` as mandatory.

Per-operation feature maps must be exact.

Required feature sets include:

- article
- tweetDetail
- search
- tweetCreate
- timeline
- bookmarks
- likes
- lists
- homeTimeline
- userTweets
- following
- explore

Required field toggles include:

- article field toggles for `TweetDetail`
- `withArticlePlainText: false` for `UserTweets`
- `withAuxiliaryUserLabels: false` for `UserByScreenName`

Missing or incorrect feature flags are considered implementation bugs.

Support runtime feature overrides from:

- `~/.config/bird/features.json`

## Critical Behavior Corrections

Implement the verified corrections from `/Users/mudrii/src/bird-docs/corrections.md`.

This is mandatory.

Especially preserve:

- response path corrections
- variable name corrections
- `DeleteRetweet` requires both `tweet_id` and `source_tweet_id`
- follow/unfollow is REST-first, GraphQL fallback last
- `getCurrentUser` tries four API endpoints plus dual HTML fallback
- cursor extraction checks `content.cursorType` directly
- `is_blue_verified` is at the top level of the user result
- pagination delay is before fetch, not after
- media upload is chunked at exactly 5 MB
- bookmark folder retries without `count` when required
- `withCommunity: true` on Home timelines
- `includePromotedContent: false` on `UserTweets`
- hard `UserTweets` limit of 10 pages
- following/followers REST fallback only after refreshed 404 failure
- bookmark retry policy and backoff values
- article text has priority before note tweet text
- Draft.js atomic entity mapping
- media poll default delay of 2 seconds
- bookmarks extra variables
- bookmark folder `includePromotedContent: true`
- exact fatal strings for `UserTweets`
- unwrap tweet results via `result.tweet` when present
- 5-path tweet result collection
- news headline field preference

## Tweet Parsing And Normalization

Use `/Users/mudrii/src/bird-docs/response-parsing.md`.

Required parsing behavior includes:

- timeline instruction parsing
- cursor extraction
- tweet normalization
- quote depth behavior
- user parsing
- media parsing
- article rendering
- note tweet extraction
- Draft.js content rendering

Tweet text priority must be:

1. article plain text and article-derived content
2. note tweet text
3. legacy full text

Do not reverse this priority.

## Pagination

Preserve the reference pagination semantics exactly.

There are two distinct paginator patterns with different stop conditions. Do not unify them.

**Generic `paginateCursor`** — used ONLY by `getRepliesPaged` and `getThreadPaged`:

- delay before fetch, skip the first page
- stop ONLY when cursor is empty or unchanged
- does NOT stop on empty page or zero new items
- on error with accumulated items: return `{success: false, error, items, nextCursor: currentCursor}` where `nextCursor` is the CURRENT cursor (not the next one)

**Inline paginators** — used by Bookmarks, Likes, BookmarkFolderTimeline, Search, Home, UserTweets, ListTimeline:

- delay before fetch, skip the first page
- stop when cursor is empty or unchanged OR `page.tweets.length === 0` OR `added === 0`
- same partial-result-on-error behavior as above

Do not normalize all paginators into a single simplified rule set.

## Retry And Fallback Behavior

Implement the real fallback logic, including:

- GraphQL 404 query ID refresh and one retry
- CreateTweet error code `226` fallback to `POST /i/api/1.1/statuses/update.json`
- bookmark retry behavior on `429`, `500`, `502`, `503`, `504`
- `Retry-After` support
- REST-first follow/unfollow strategy
- following/followers REST fallback
- `UserByScreenName` REST fallback after hardcoded GraphQL IDs fail

Do not add broad automatic retries where the reference implementation does not retry.

## Media Upload

Implement the chunked upload flow documented in the source:

1. INIT
2. APPEND in exactly 5 MB chunks
3. FINALIZE
4. STATUS polling when processing is asynchronous
5. optional alt-text metadata call

Rules:

- max STATUS poll attempts: 20
- default poll delay fallback: 2 seconds
- alt text only for `image/*`
- support documented MIME types and category mapping
- preserve upload header differences between upload and JSON metadata calls

## Output Formatting

Replicate the reference output behavior:

- color output on TTY
- `--no-color`
- `NO_COLOR=1`
- emoji prefixes
- `--no-emoji`
- `--plain`
- OSC 8 hyperlinks where appropriate
- `--json`
- `--json-full`

The Go version may use a different implementation internally, but the user-visible behavior should match.

## Error Handling

Errors must be explicit, contextual, and non-leaky.

Rules:

- never include secrets in errors
- preserve meaningful HTTP and GraphQL error information
- validation and usage errors should map to exit code `2`
- runtime/API/auth/network errors should map to exit code `1`
- partial pagination results should surface both items and error state when the reference does; exit code is `1`

## Type Safety

Strongly prefer concrete structs over `map[string]any` or `interface{}`.

Allowed exceptions:

- highly dynamic raw `_raw` response payloads
- feature override documents
- narrowly scoped decoding helpers when shape is truly unbounded

Even when dynamic decoding is needed, isolate it at the edge and convert to typed structs quickly.

## Logging

Use `log/slog` or an equivalent structured logger.

Rules:

- default logging should be quiet
- support info/debug verbosity if you intentionally add it
- redact all credential-bearing headers and cookies
- never print auth secrets

If you add logging flags not present in the reference CLI, document them as non-parity extensions.

## Dependencies

Prefer the standard library where practical.

Reasonable dependencies:

- CLI framework: use **Cobra** (`github.com/spf13/cobra`). The CLI surface (20+ commands, 30+ flags, per-command flags, shorthand detection) is too large for stdlib `flag`.
- `github.com/google/uuid`
- JSON5 parser: use `github.com/tailscale/hujson` (minimal, well-maintained near-JSON5 superset). Config files are `.json5` so a parser is required.
- SQLite driver for browser cookie access: use `modernc.org/sqlite` (pure Go, no CGo required)

Avoid framework-heavy stacks.

Do not add dependencies unless they solve a real parity problem.

## Testing Strategy

TDD and ATDD are mandatory.

### Unit Tests

Write tests first for:

- tweet ID extraction
- list ID extraction
- handle normalization
- tweet input detection
- tweet normalization
- cursor extraction
- user parsing
- media parsing
- Draft.js rendering
- feature map generation
- field toggle generation
- header building
- request builders
- query ID store behavior
- refresh fallback behavior
- pagination behavior
- bookmark thread filtering
- error mapping

Use:

- table-driven tests
- fuzz tests where input parsing matters
- property-based tests where transformations benefit from invariants

### Integration Tests

Use a mocked HTTP server and JSON fixtures to test:

- auth flows
- get tweet
- replies/thread pagination
- search
- home timelines
- bookmarks
- likes
- user tweets
- user lookup
- current user
- lists
- news
- media upload
- follow/followers fallbacks
- query ID refresh and retry
- CreateTweet fallback to `statuses/update.json`

### Fixtures

Store fixtures in `tests/fixtures/`.

Build fixtures from the reference docs and captured shapes in the reference source.

Include edge cases such as:

- empty results
- article tweets
- note tweets
- quoted tweets
- media tweets
- cursor changes
- no cursor
- repeated cursor
- `GRAPHQL_VALIDATION_FAILED`
- 404 query ID invalidation
- 429 bookmark retry
- `UserUnavailable`
- user suspended / user not found

## Quality Gates

Before considering work complete, require:

- `go test ./...`
- `go test -race ./...`
- coverage for core logic
- `go vet ./...`
- `golangci-lint run ./...`
- reproducible build of `./cmd/bird`

CI must run all of the above.

## Makefile

Provide at least:

- `test`
- `test-race`
- `lint`
- `vet`
- `build`
- `clean`

Optional but useful:

- `coverage`
- `fmt`
- `ci`

The `build` target must embed version and git SHA using ldflags:

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  = -ldflags "-X main.version=$(VERSION) -X main.gitSHA=$(GIT_SHA)"

build:
	go build $(LDFLAGS) -o bin/bird ./cmd/bird
```

`bird --version` must output `<version> (<gitSHA>)`, e.g. `0.1.0 (3df7969b)`.

## CI

Use GitHub Actions or equivalent to run:

- checkout
- setup Go 1.23
- `go mod tidy` or verification step
- `go mod verify`
- tests
- race tests
- vet
- lint
- build

## Implementation Sequence

Implement in this order:

1. repository scaffolding
2. types and extract/normalize helpers
3. auth and config resolution
4. HTTP client, headers, request builders
5. query ID store and refresh logic
6. feature maps and field toggles
7. tweet parsing and normalization
8. read operations
9. current user and user lookup flows
10. timelines, lists, bookmarks, likes, news
11. media upload
12. write operations and fallbacks
13. CLI output formatting
14. CLI commands and exit codes
15. full integration coverage

## Definition Of Done

The implementation is done only when all of the following are true:

- the Go binary reproduces the actual `bird` CLI surface
- JSON output matches the reference behavior closely enough to act as a drop-in replacement
- all documented corrections are implemented
- query ID refresh behavior works
- CreateTweet REST fallback works
- `UserByScreenName` hardcoded ID rotation works
- auth resolution order works
- bookmark thread expansion behavior works
- media upload works
- tests and lint pass
- the implementation lives entirely in `/Users/mudrii/src/gobird`

## Final Instruction

When uncertain, do not guess.

Inspect:

- `/Users/mudrii/src/bird`
- `/Users/mudrii/src/bird-docs`

Then implement the behavior exactly, with tests first.
