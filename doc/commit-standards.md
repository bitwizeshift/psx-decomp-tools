# Commit Standards

This project requires all contributions and commits to include the same
standardized language and trailers in the commit message. This ensures a
consistent and clear history which enables clear automation and easy
navigation.

## General Requirements

* Commits should be small, granular, and easy to follow and revert. Ideally,
  the same SOC practices that would be applied to software development should
  be applied to commits; each commit identifies a separtion of concerns.

### Commit Titles

* must be in imperitive, present-tense

* must be no longer than 50 characters

* must summarize the change being performed

### Commit Messages

* Must not exceed 72-character wide

  * Exceptions are made if a message contains a link or other figure that
    exceeds the 72 character rule by nature

* Must indicate the rationale for the change, what was changed, and why

  * In general, more details are always better to help identify the cause of
    changes in a repository

* Should trailer meta-information, as described in [Trailers](#trailers)

## Trailers

### `Change-Category`

> [!NOTE]
>
> This trailer is **required** for all commits

`Change-Category` is a Git Trailer that must be added to commit messages to
categorize the type of change being performed.

The following categories are supported:

* `feature` - A commit that adds a new functionality or capability to the
  codebase.

* `bugfix` - A commit that fixes a bug or defect in the codebase.

* `security` - A commit that either hardens security, or addresses a present
  security vulnerability in the codebase.

* `internal` - A commit that does not directly affect product functionality
  (refactoring, tooling, tests, CI/CD, dependency management, build system
  changes, etc.).

* `deprecation` - A commit that deprecates a feature, function, or other element
  of the codebase.

### `Change-Impact`

> [!NOTE]
>
> This trailer is **required** for all commits

`Change-Impact` is a Git Trailer that must be added to commit messages to
indicate the level of impact of the change being performed. This is necessary
for determining backwards-compatibility for versioning.

The following impact levels are supported:

* `none` - A commit that does not modify existing behavior or add new
  functionality. **No observable impact** on users of the codebase.

* `corrective` - A commit that makes a backwards-compatible correction of
  existing behavior. This includes bug fixes, security patches, and other
  changes that do not add new functionality, or modify existing functionality in
  a way that would require users to update their code or workflows.

  This _may_ be observable if the corrective action is correcting a **documented**
  behavior that is not working as documented, but it should not be observable if
  the corrective action is correcting an **undocumented** behavior.

* `additive` - A commit that adds new functionality or capabilities without
  modifying existing behavior. This includes new features, optimizations, and
  other improvements that do not change the expected behavior of existing code.

* `breaking` - A commit that modifies existing behavior in a way that may cause
  issues for users of the codebase. This includes changes to public APIs, changes
  to expected behavior of existing code, and other modifications that may require
  users to update their code or workflows.

### `Component`

`Component` is a Git Trailer that may be added to commit messages to indicate
the component(s) of the codebase that are affected by the change being performed.
This trailer may be repeated more than once to indicate multiple components.

The `Component` field should always be 1 of:

1. `infra` - a special-case component indicating that the commit is related to
   infrastructure, CI/CD, or build system changes.

2. The name of a subdirectory under `cmd`, indicating that it's a commit touching
   a specific CLI tool. E.g. `cmd/dump-image` would be `Component: dump-image`

3. The name of a subdirectory under `internal`, indicating that it's a commit
   touching a specific internal package component. E.g. `internal/disc/cue`
   would be `Component: disc`

### `Fixes`

The `Fixes` Git Trailer is used to indicate that a commit resolves an issue
tracked in the project's issue tracker.

This borrows from GitHub's automatic issue closing syntax. The format is as
follows:

```text
Fixes: #<issue-number>
```

### `Co-authored-by`

The `Co-authored-by` Git Trailer is used to indicate that a commit was
co-authored by multiple contributors. This is frequently used to give credit
to users that may have contributed useful ideas, information, or other help
towards a change in the codebase.

If a contributor does not want their email listed publicly, the GitHub alias of
`<username>@users.noreply.github.com` can be used to mask the email while still
giving proper credit.

The format is as follows:

```text
Co-authored-by: NAME <EMAIL>
```

For example:

```text
Co-authored-by: Matthew Rodusek <bitwizeshift@users.noreply.github.com>
```
