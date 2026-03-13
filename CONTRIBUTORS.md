# Contributing to gobird

gobird is a Go CLI tool and library that wraps the undocumented Twitter/X GraphQL and REST APIs. The API is reverse-engineered: response shapes change without notice, query IDs rotate, and feature flag maps must be cloned before use. This guide exists to prevent rework caused by the sharp edges in the codebase.

Read this document fully before writing code. Most contributor mistakes fall into a small set of well-known categories covered here.

---

## Before you start

Read these files first. They contain decisions that are not obvious from the code alone:

| File | Why you need it |
|---|---|
| `docs/agent-context.md` | The 86+ API corrections — specific facts the codebase relies on that contradict naive assumptions |
| `docs/architecture.md` | Package responsibilities, data flow, and design constraints |
| `docs/development-guide.md` | Full setup walkthrough, tool versions, and test conventions |

### The 10 rules that prevent most rework

1. Clone feature flag maps before modifying them — never mutate a package-level map directly.
2. Use `cachedUserID()` to access the authenticated user's ID — never read `c.userID` directly.
3. Use `cmd.Context()` in CLI command handlers — never `context.Background()`.
4. Register every new `cobra.Command` in `internal/cli/root.go` under `NewRootCmd`.
5. Add new operations to **all three** query ID maps: `FallbackQueryIDs`, `BundledBaselineQueryIDs`, and `PerOperationFallbackIDs` in `internal/client/constants.go`.
6. Export new public types via `pkg/bird/types.go` as type aliases — do not expose `internal/` types directly.
7. Error strings must be lowercase and wrapped with context (e.g. `fmt.Errorf("search: %w", err)`).
8. Compile regular expressions at package level (`var re = regexp.MustCompile(...)`) — never inside a function.
9. Handle all `Close()` errors — `golangci-lint` errcheck will fail the CI if you ignore them.
10. Write table-driven tests and run `go test -race ./...` before opening a PR.

---

## Development setup

```sh
git clone https://github.com/mudrii/gobird.git
cd gobird

# Build the binary
make build

# Run tests
make test

# Run tests with race detector
make test-race

# Run vet + tests + race (mirrors CI)
make ci

# Lint (requires golangci-lint)
make lint
```

Install tools:

```sh
# golangci-lint
brew install golangci-lint
# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

The binary is written to `bin/gobird`.

---

## The 86 API Corrections

The Twitter/X GraphQL API is undocumented and was reverse-engineered from browser traffic. The `docs/agent-context.md` file records 86+ corrections — cases where an obvious implementation approach is wrong because of a quirk in the real API response.

These corrections exist because:
- Fields appear at different nesting levels than expected (e.g. `is_blue_verified` is on the top-level `result`, not inside `legacy`)
- Some operations silently return partial data rather than errors
- Query ID sets differ across API contexts
- Feature flag maps must be sent as copies, not references

### Corrections every contributor must know before touching API code

| Correction | What it means |
|---|---|
| #5 | `UserByScreenName` never uses the runtime cache — it has hardcoded-only IDs in `PerOperationFallbackIDs` |
| #8 | `is_blue_verified` comes from `result.is_blue_verified`, not `result.legacy.is_blue_verified` |
| #31 | `PreviewURL` is set for **any** media that has `sizes.small`, not just video |
| #32 | Media dimensions use `sizes.large` first, `sizes.medium` as fallback — not the top-level `original` |
| #46 | Default news tabs are `forYou`, `news`, `sports`, `entertainment` — **not** `trending` |
| #71 | `FallbackQueryIDs` has exactly 29 entries — adding a new operation requires updating the comment too |

When you encounter a response shape that surprises you, check `docs/agent-context.md` before assuming the code is wrong.

---

## Code organisation rules

### `internal/` vs `pkg/`

| Location | Contains |
|---|---|
| `pkg/bird/` | The public API surface: `Client`, all method wrappers, `ResolveCredentials`, exported types. Nothing else. |
| `internal/client/` | The actual HTTP client, GraphQL request construction, pagination, query ID management. |
| `internal/cli/` | Cobra command definitions. One file per command group. No business logic. |
| `internal/auth/` | Credential resolution and browser cookie extraction. |
| `internal/config/` | JSON5 config loading and env var overlay. |
| `internal/types/` | All normalised output types (`TweetData`, `TwitterUser`, etc.) and option structs. |
| `internal/parsing/` | Response JSON parsing: tweet extraction, cursor extraction, list/news parsing. |
| `internal/output/` | Formatting and JSON serialisation for CLI output. |
| `internal/testutil/` | Shared test helpers (golden file support). |

### What goes where

- New CLI commands: add a file in `internal/cli/` and register in `root.go`.
- New API operations: implement in `internal/client/`, wrap in `pkg/bird/client_methods.go`.
- New public types: define in `internal/types/models.go` or `options.go`, re-export in `pkg/bird/types.go`.
- New parsing logic: goes in `internal/parsing/`, tested with unit tests against fixture JSON.
- New output formatters: goes in `internal/output/`.

### Naming conventions

- Files: `snake_case.go`
- Types: `PascalCase`
- Functions and methods: `camelCase` (unexported) or `PascalCase` (exported)
- Test files: `<subject>_test.go` in the same package (white-box) or `_test` package (black-box)
- Golden fixtures: `testdata/<operation>.json`

### Error message format

- Always lowercase: `"search: result had no items"` not `"Search: result had no items"` (staticcheck ST1005)
- Always wrap with context: `fmt.Errorf("tweet detail: %w", err)`
- Never swallow errors silently

---

## Adding new features — checklist

Before writing code:

- [ ] Read the existing implementation for the most similar feature
- [ ] Check `docs/agent-context.md` for any corrections affecting this operation
- [ ] Verify the operation's response shape in the browser with DevTools → Network

During implementation:

- [ ] Add the operation to all three query ID maps in `internal/client/constants.go`:
  - `FallbackQueryIDs` (hardcoded fallback, one entry per operation)
  - `BundledBaselineQueryIDs` (baseline from embedded bundle, if applicable)
  - `PerOperationFallbackIDs` (ordered list of IDs to try, required for every operation)
- [ ] Clone any feature map before building the GraphQL request variables
- [ ] Use `cmd.Context()` in CLI handlers, not `context.Background()`
- [ ] Use `cachedUserID()` for the authenticated user's ID, not `c.userID`
- [ ] Add the new `cobra.Command` to `NewRootCmd` in `internal/cli/root.go`
- [ ] Export any new public types via `pkg/bird/types.go`
- [ ] Write table-driven unit tests for all parsing and formatting logic
- [ ] Write mock HTTP tests for the new client method

Before opening the PR:

- [ ] `make ci` passes (vet + test + test-race + build)
- [ ] `make lint` passes with no new warnings
- [ ] `docs/` updated if the change affects architecture, the API correction log, or public behaviour

---

## Common mistakes that cause rework

These are the specific mistakes that have generated the most rework on this project. Read this list before touching existing code.

### Modifying a feature flag map without cloning it

**Wrong:**
```go
vars["features"] = featureMap  // featureMap is the package-level map
```

**Right:**
```go
features := make(map[string]bool, len(featureMap))
for k, v := range featureMap {
    features[k] = v
}
vars["features"] = features
```

Package-level maps are shared across requests. Modifying one mutates state for all concurrent callers.

### Reading `c.userID` directly

`c.userID` may be empty until the first `GetCurrentUser` call. Use the `cachedUserID()` method which lazily resolves and caches the value.

### Using `context.Background()` in CLI handlers

CLI handlers must propagate the command's context so that timeouts, cancellation, and signal handling work correctly:

**Wrong:**
```go
result := c.Search(context.Background(), query, opts)
```

**Right:**
```go
result := c.Search(cmd.Context(), query, opts)
```

### Not registering a new command in `root.go`

A `cobra.Command` that is not added to the root command via `root.AddCommand(newYourCmd())` in `NewRootCmd` will never be reachable. The command compiles and the tests may pass, but `gobird your-command` will return "unknown command".

### Not adding to all three query ID maps

Every new GraphQL operation needs an entry in:
1. `FallbackQueryIDs` — the last-resort hardcoded ID
2. `BundledBaselineQueryIDs` — the baseline from the embedded JS bundle (skip only if the operation has no bundle entry)
3. `PerOperationFallbackIDs` — the ordered list of IDs the client tries in sequence

Missing any one of these causes the operation to silently use a wrong ID or skip fallback attempts, producing `401` or `400` errors that are hard to diagnose.

### Error strings starting with a capital letter

staticcheck rule ST1005: error strings must not be capitalised.

**Wrong:** `errors.New("Tweet not found")`
**Right:** `errors.New("tweet not found")`

### Ignoring `Close()` errors

`golangci-lint` errcheck catches unclosed resources. The pattern `defer f.Close()` must be `defer f.Close() //nolint:errcheck` only when the error genuinely cannot be handled. For response bodies, always drain and close:

```go
defer func() {
    _, _ = io.Copy(io.Discard, resp.Body)
    if err := resp.Body.Close(); err != nil {
        // handle or log
    }
}()
```

### Compiling regex inside functions

**Wrong:**
```go
func parseID(s string) string {
    re := regexp.MustCompile(`^\d+$`)  // compiled on every call
    ...
}
```

**Right:**
```go
var idRe = regexp.MustCompile(`^\d+$`)  // compiled once at package init

func parseID(s string) string {
    ...
}
```

---

## Pull request process

### Before opening a PR

- [ ] `make ci` passes locally
- [ ] `make lint` passes with no new warnings
- [ ] New API code has mock HTTP tests
- [ ] New parsing code has unit tests
- [ ] `go test -race ./...` passes
- [ ] No real network calls in tests
- [ ] Commit messages are imperative, lowercase, under 72 characters (e.g. `add list-timeline command`)
- [ ] `docs/` updated if architecture, the correction log, or public behaviour changed

### What reviewers check

1. All three query ID maps updated for new operations
2. Feature maps cloned before modification
3. `cmd.Context()` used in CLI handlers
4. Error strings lowercase and wrapped with context
5. No `context.Background()` in reachable paths
6. New commands registered in `root.go`
7. New public types exported via `pkg/bird/types.go`
8. Race detector passes
9. No real network calls in tests

### Run full CI locally

```sh
make ci
```

This runs: `go vet ./...`, `go test ./...`, `go test -race ./...`, `make build`.

Run lint separately (it requires `golangci-lint` to be installed):

```sh
make lint
```

---

## Testing requirements

- All new client methods need mock HTTP tests. Use `httptest.NewServer` with fixture JSON responses stored in `testdata/`.
- All new parsing functions need unit tests with table-driven cases covering at least: normal input, empty input, and malformed input.
- The race detector must pass: `go test -race ./...`. Concurrent map access will fail here if you forgot to clone a shared map.
- No real network calls in tests. Tests must pass offline. Any test that calls `x.com` will be rejected.
- Use golden files for complex output assertions. The `internal/testutil` package provides golden file support.

---

## Wire protocol changes

The Twitter/X GraphQL API is **undocumented and changes without notice**. Query IDs rotate when Twitter deploys new JavaScript bundles. Response field paths move. Feature flag requirements change.

### Symptoms of a stale query ID

- HTTP 400 or 403 responses from GraphQL endpoints
- Responses with shape `{"errors": [{"message": "..."}]}` and no `data` field
- `query-ids` command shows IDs that differ from what the browser sends

### How to update query IDs

1. Open x.com in a browser with DevTools → Network tab open.
2. Filter to `graphql` requests and perform the action (search, load timeline, etc.).
3. Find the request URL: `https://x.com/i/api/graphql/<queryID>/<OperationName>`.
4. Copy the new query ID.
5. Update **all three** maps in `internal/client/constants.go`:
   - Set the new ID as the first entry in `PerOperationFallbackIDs[operationName]`
   - Update `BundledBaselineQueryIDs[operationName]`
   - Update `FallbackQueryIDs[operationName]` (the last-resort fallback)
6. Keep the old ID as a secondary entry in `PerOperationFallbackIDs` so existing cached states continue to work during rollout.
7. Run `make ci` to confirm the change does not break existing tests.

### How to verify response paths

1. Capture a real response with `--json-full` and examine `_raw`.
2. Trace the parsing path in `internal/parsing/` against the captured response.
3. When a field moves, update the parsing code **and** add a new entry to `docs/agent-context.md` with the correction number incremented.

Do not assume a field is at the path it was yesterday. Always verify against a fresh browser capture before filing a bug.
