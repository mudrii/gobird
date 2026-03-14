# Contributing

Thanks for contributing to `gobird`.

## Development setup

Requirements:
- Go 1.24+
- `golangci-lint`

Common commands:

```sh
make build
make test
make test-race
make lint
make ci
```

## Before opening a pull request

Please make sure:
- tests pass
- lint passes
- docs stay aligned with flags, commands, env vars, and runtime behavior
- new behavior includes tests where practical

## Pull request guidance

- keep changes scoped
- include a short problem statement and approach summary
- call out user-facing changes in the PR description
- update `CHANGELOG.md` for notable user-facing changes

## Reporting bugs

Open an issue with:
- the command you ran
- the exact error output
- OS / architecture
- whether auth came from flags, env vars, Safari, Chrome, or Firefox

Security-sensitive issues should go through the process in [SECURITY.md](SECURITY.md).
