#!/usr/bin/env bash
#
# Validates that a commit message conforms to the project's commit standards,
# as described in doc/commit-standards.md.
#
# It enforces the required `Change-Category` and `Change-Impact` trailers, and
# validates the optional, repeatable `Component` trailer against the set of
# components that actually exist in the repository.
#
# Usage:
#   validate-commit-trailers.sh [<commit>...]
#       Validate one or more commits. A `<commit>` may be a single revision or a
#       `A..B` range (expanded via `git rev-list`). Defaults to HEAD.
#
#   validate-commit-trailers.sh --message-file <path>
#       Validate a raw commit message read from <path>. Use `-` to read from
#       stdin. This is the mode used by the `commit-msg` git hook and CI.

set -euo pipefail

# Allowed values for the required trailers. Keep these in sync with
# doc/commit-standards.md.
readonly ALLOWED_CATEGORIES=(feature bugfix security internal deprecation)
readonly ALLOWED_IMPACTS=(none corrective additive breaking)

# Resolve the repository root so component discovery and tooling work regardless
# of the caller's current working directory.
REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT

# Prints the program usage to stderr.
function usage {
  cat >&2 <<'EOF'
Usage:
  validate-commit-trailers.sh [<commit>...]
  validate-commit-trailers.sh --message-file <path|->
EOF
}

# Returns success if the first argument is contained in the remaining arguments.
#
#   contains <needle> <haystack>...
function contains {
  local needle="${1}"
  shift
  local item
  for item in "$@"; do
    if [[ "${item}" == "${needle}" ]]; then
      return 0
    fi
  done
  return 1
}

# Echoes the set of valid `Component` values, one per line. This is derived
# dynamically from the repository layout so it never goes stale:
#   - the special-case `infra` component
#   - every first-level subdirectory of cmd/
#   - every first-level subdirectory of internal/
function valid_components {
  echo "infra"

  local dir parent
  for parent in "${REPO_ROOT}/cmd" "${REPO_ROOT}/internal"; do
    [[ -d "${parent}" ]] || continue
    for dir in "${parent}"/*/; do
      # Guard against a non-matching glob expanding to the literal pattern.
      [[ -d "${dir}" ]] || continue
      basename "${dir}"
    done
  done
}

# Validates a single commit message provided on stdin. `label` is a
# human-readable identifier (short SHA or "(message file)") used in diagnostics.
#
# Prints one error line per violation and returns non-zero if any are found.
function validate_message {
  local label="${1}"
  local message
  message="$(cat)"

  # `git interpret-trailers --parse` normalises folding/spacing and emits the
  # trailers as `Key: value` lines.
  local trailers
  trailers="$(printf '%s\n' "${message}" | git interpret-trailers --parse)"

  local -a categories=() impacts=() components=()
  local key value line
  while IFS= read -r line; do
    [[ -n "${line}" ]] || continue
    key="${line%%:*}"
    value="${line#*:}"
    # Trim surrounding whitespace from the value.
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"

    # Trailer keys are matched case-insensitively.
    case "${key,,}" in
      change-category) categories+=("${value}") ;;
      change-impact) impacts+=("${value}") ;;
      component) components+=("${value}") ;;
    esac
  done <<<"${trailers}"

  local -a errors=()

  # Change-Category: required, exactly once, from the allowed set.
  if [[ "${#categories[@]}" -eq 0 ]]; then
    errors+=("missing required trailer 'Change-Category'")
  elif [[ "${#categories[@]}" -gt 1 ]]; then
    errors+=("trailer 'Change-Category' must appear exactly once (found ${#categories[@]})")
  elif ! contains "${categories[0]}" "${ALLOWED_CATEGORIES[@]}"; then
    errors+=("invalid 'Change-Category' value '${categories[0]}' (allowed: ${ALLOWED_CATEGORIES[*]})")
  fi

  # Change-Impact: required, exactly once, from the allowed set.
  if [[ "${#impacts[@]}" -eq 0 ]]; then
    errors+=("missing required trailer 'Change-Impact'")
  elif [[ "${#impacts[@]}" -gt 1 ]]; then
    errors+=("trailer 'Change-Impact' must appear exactly once (found ${#impacts[@]})")
  elif ! contains "${impacts[0]}" "${ALLOWED_IMPACTS[@]}"; then
    errors+=("invalid 'Change-Impact' value '${impacts[0]}' (allowed: ${ALLOWED_IMPACTS[*]})")
  fi

  # Component: optional, repeatable, each from the dynamic set.
  if [[ "${#components[@]}" -gt 0 ]]; then
    local -a allowed_components=()
    mapfile -t allowed_components < <(valid_components)
    local component
    for component in "${components[@]}"; do
      if ! contains "${component}" "${allowed_components[@]}"; then
        errors+=("invalid 'Component' value '${component}' (allowed: ${allowed_components[*]})")
      fi
    done
  fi

  if [[ "${#errors[@]}" -gt 0 ]]; then
    local gh_error_prefix=""
    if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
      gh_error_prefix="::error::"
    fi

    local error
    for error in "${errors[@]}"; do
      echo "${gh_error_prefix}  ${label}: ${error}" >&2
    done
    return 1
  fi
  return 0
}

function main {
  local mode="commits"
  local message_file=""
  local -a refs=()

  while [[ "$#" -gt 0 ]]; do
    case "${1}" in
      --message-file)
        mode="message-file"
        message_file="${2:-}"
        if [[ -z "${message_file}" ]]; then
          echo "error: --message-file requires a path argument" >&2
          usage
          return 2
        fi
        shift 2
        ;;
      -h | --help)
        usage
        return 0
        ;;
      --)
        shift
        refs+=("$@")
        break
        ;;
      -*)
        echo "error: unknown option '${1}'" >&2
        usage
        return 2
        ;;
      *)
        refs+=("${1}")
        shift
        ;;
    esac
  done

  local failed=0

  if [[ "${mode}" == "message-file" ]]; then
    if [[ "${message_file}" == "-" ]]; then
      validate_message "(message file)" || failed=1
    else
      validate_message "(message file)" <"${message_file}" || failed=1
    fi
  else
    # Default to HEAD when no refs are supplied.
    if [[ "${#refs[@]}" -eq 0 ]]; then
      refs=(HEAD)
    fi

    # Expand any `A..B` ranges into individual commits while preserving the
    # order in which refs were given.
    local -a commits=()
    local ref
    for ref in "${refs[@]}"; do
      if [[ "${ref}" == *..* ]]; then
        local sha
        while IFS= read -r sha; do
          [[ -n "${sha}" ]] && commits+=("${sha}")
        done < <(git rev-list --reverse "${ref}")
      else
        commits+=("${ref}")
      fi
    done

    local commit short
    for commit in "${commits[@]}"; do
      short="$(git rev-parse --short "${commit}")"
      git show -s --format=%B "${commit}" \
        | validate_message "${short}" || failed=1
    done
  fi

  if [[ "${failed}" -ne 0 ]]; then
    echo "error: one or more commits do not conform to doc/commit-standards.md" >&2
    return 1
  fi

  echo "All checked commit message(s) conform to the commit standards."
  return 0
}

main "$@"
