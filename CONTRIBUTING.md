# Contributing

See [README.md](README.md) for setup and development commands.

This file is the source of truth for the contribution workflow.

Before making any code change, read [ARCHITECTURE.md](ARCHITECTURE.md).

## Branches and commits

Branch from `main`. Keep one concern per branch.

Name branches `<type>/<kebab-slug>`, where the type matches the Conventional Commits type set — for example `feat/hw-large-view` or `fix/sidebar-icon-tweaks`. Current branches follow this, so the prefix keeps history scannable.

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, and `perf`. Use `!` or a `BREAKING CHANGE:` footer for breaking changes.

## Pull requests

- Keep one concern per pull request.
- Explain what changed, why, and any important design decisions.
- Follow [STYLE.md](STYLE.md) and run the verification commands in [AGENTS.md](AGENTS.md).
- Address review feedback before merge.

Before submitting:

- [ ] Tests added or updated for changed behavior
- [ ] Verification commands pass
- [ ] Documentation updated when behavior, commands, or public interfaces changed

## Issues

- [Report a bug](https://github.com/markmybytes/install-it/issues/new?labels=bug) — include versions, environment details, and a minimal reproduction.
- [Request a feature](https://github.com/markmybytes/install-it/issues/new?labels=enhancement) — describe the problem, not just the solution.
