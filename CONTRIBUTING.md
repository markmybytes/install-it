# Contributing

See [README.md](README.md) for setup and development commands.

## Before You Start

- Check existing issues and pull requests before opening a new one.
- Discuss substantial feature changes before implementation.
- Branch from the repository's default branch.

## Branches and Commits

Use a descriptive branch name. Keep each branch focused on one concern.

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, and `perf`. Use `!` or a `BREAKING CHANGE:` footer for breaking changes.

## Pull Requests

- Keep one concern per pull request.
- Explain what changed, why, and any important design decisions.
- Add or update tests for behavior changes and bug fixes.
- Update documentation when behavior, commands, or public interfaces change.
- Run the verification commands from [README.md](README.md) and follow [STYLE.md](STYLE.md) before pushing.
- Address review feedback before merge.

## Issues

- [Report a bug](https://github.com/markmybytes/install-it/issues/new?labels=bug&template=bug-report---.md) — include versions, environment details, and a minimal reproduction.
- [Request a feature](https://github.com/markmybytes/install-it/issues/new?labels=enhancement&template=feature-request---.md) — describe the problem, not just the solution.
