# Contributing to hop

Thanks for taking the time to contribute. This document explains how to report issues, propose changes, and the requirements your contribution must meet to be merged.

## Reporting bugs and requesting features

Open a [GitHub issue](https://github.com/philippe-desplats/hop/issues). For a bug, include your OS and architecture, the output of `hop version`, the exact command you ran, and what you expected versus what happened. For security issues, do not open a public issue: follow [SECURITY.md](SECURITY.md) instead.

## Proposing a change

1. Fork the repository and create a branch from `master`.
2. Make your change, with tests for any new behavior or bug fix.
3. Make sure the full check suite passes locally (see below).
4. Open a pull request describing what changed and why. Link the issue it addresses if there is one.

Small fixes (typos, docs, obvious bugs) can go straight to a pull request. For larger changes, open an issue first to discuss the direction before investing time.

## Requirements for acceptable contributions

A contribution is merged only once it meets all of the following:

- **Formatting**: code is formatted with `gofmt` (`go fmt ./...` leaves no diff).
- **Linting**: `golangci-lint run ./...` passes with no new issues, using the project's [`.golangci.yml`](.golangci.yml). Do not silence findings with blanket `//nolint` directives; if a suppression is genuinely needed, scope it to the specific linter and justify it in a comment.
- **Vetting**: `go vet ./...` is clean.
- **Tests**: `go test ./...` passes, and new behavior or bug fixes ship with tests.
- **Commits**: messages follow [Conventional Commits](https://www.conventionalcommits.org/), for example `feat(hub): add pin shortcut` or `fix(resolver): handle empty keyword`. Allowed types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `revert`.
- **Cross-platform**: hop targets macOS and Linux on amd64 and arm64. Keep platform-specific code behind build tags or runtime checks, and never assume a macOS-only tool is present.
- **No AI or vendor attribution** in commits, code comments, or documentation.

## Local development

```sh
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

To try your build without touching your real index, use the self-contained playground:

```sh
go build -o /tmp/hop ./cmd/hop
PATH="/tmp:$PATH" source sample/demo.sh
```

## License

By contributing, you agree that your contributions are licensed under the project's [MIT License](LICENSE).
