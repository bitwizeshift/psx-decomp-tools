#!/usr/bin/env bash
#
# Installs the project's local git hooks. Currently this installs a `commit-msg`
# hook that validates commit messages against doc/commit-standards.md using
# scripts/validate-commit-trailers.sh.
#
# Usage:
#   scripts/install-git-hooks.sh

set -euo pipefail

# A marker written into hooks installed by this script so we can recognise (and
# safely overwrite) our own hooks while preserving any pre-existing ones.
readonly HOOK_MARKER="# installed-by: scripts/install-git-hooks.sh"

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT

HOOKS_DIR="$(git rev-parse --git-path hooks)"
readonly HOOKS_DIR

# Installs the commit-msg hook, backing up any unmanaged hook that already
# exists so we never clobber a user's own hook.
function install_commit_msg_hook {
  local hook_path="${HOOKS_DIR}/commit-msg"

  if [[ -e "${hook_path}" ]] && ! grep -qF "${HOOK_MARKER}" "${hook_path}"; then
    local backup="${hook_path}.bak"
    echo "warning: existing commit-msg hook found; backing it up to ${backup}" >&2
    mv "${hook_path}" "${backup}"
  fi

  mkdir -p "${HOOKS_DIR}"
  cat >"${hook_path}" <<EOF
#!/usr/bin/env bash
${HOOK_MARKER}
#
# Validates the commit message against doc/commit-standards.md.
set -euo pipefail

repo_root="\$(git rev-parse --show-toplevel)"
exec "\${repo_root}/scripts/validate-commit-trailers.sh" --message-file "\${1}"
EOF
  chmod +x "${hook_path}"

  echo "Installed commit-msg hook at ${hook_path}"
}

function main {
  # Ensure the validator we delegate to is present and executable.
  local validator="${REPO_ROOT}/scripts/validate-commit-trailers.sh"
  if [[ ! -x "${validator}" ]]; then
    echo "error: expected validator at ${validator} to be executable" >&2
    return 1
  fi

  install_commit_msg_hook
  echo "Git hooks installed successfully."
  return 0
}

main "$@"
