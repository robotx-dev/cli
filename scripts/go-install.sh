#!/usr/bin/env bash
set -euo pipefail

GO_PACKAGE="${ROBOTX_GO_PACKAGE:-github.com/robotx-dev/cli/cmd/robotx@latest}"
INSTALL_DIR="${ROBOTX_INSTALL_DIR:-}"
AUTO_PATH="${ROBOTX_AUTO_PATH:-1}"
LEGACY_GO_PACKAGE="${ROBOTX_LEGACY_GO_PACKAGE:-github.com/robotx-dev/cli@latest}"

if ! command -v go >/dev/null 2>&1; then
  echo "go is required" >&2
  exit 1
fi

if [[ -z "${INSTALL_DIR}" ]]; then
  GOBIN_VALUE="$(go env GOBIN)"
  if [[ -n "${GOBIN_VALUE}" ]]; then
    INSTALL_DIR="${GOBIN_VALUE}"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

mkdir -p "${INSTALL_DIR}"

echo "Installing ${GO_PACKAGE} to ${INSTALL_DIR}..."
if ! GOBIN="${INSTALL_DIR}" go install "${GO_PACKAGE}"; then
  if [[ "${GO_PACKAGE}" == "github.com/robotx-dev/cli/cmd/robotx@latest" ]]; then
    echo "Primary package install failed, trying legacy package ${LEGACY_GO_PACKAGE}..."
    GOBIN="${INSTALL_DIR}" go install "${LEGACY_GO_PACKAGE}"
  else
    exit 1
  fi
fi

if [[ -x "${INSTALL_DIR}/robotx_cli" && ! -e "${INSTALL_DIR}/robotx" ]]; then
  ln -sf "${INSTALL_DIR}/robotx_cli" "${INSTALL_DIR}/robotx"
fi

detect_profile_file() {
  local shell_name
  shell_name="$(basename "${SHELL:-}")"

  case "${shell_name}" in
    bash)
      if [[ "${OSTYPE:-}" == darwin* ]]; then
        echo "${HOME}/.bash_profile"
      elif [[ -f "${HOME}/.bash_profile" ]]; then
        echo "${HOME}/.bash_profile"
      else
        echo "${HOME}/.bashrc"
      fi
      ;;
    zsh)
      echo "${HOME}/.zshrc"
      ;;
    *)
      if [[ -f "${HOME}/.profile" ]]; then
        echo "${HOME}/.profile"
      else
        echo "${HOME}/.bash_profile"
      fi
      ;;
  esac
}

if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  if [[ "${AUTO_PATH}" == "1" ]]; then
    PROFILE_FILE="$(detect_profile_file)"
    PATH_EXPORT="export PATH=\"${INSTALL_DIR}:\$PATH\""
    PROFILE_DIR="$(dirname "${PROFILE_FILE}")"

    if mkdir -p "${PROFILE_DIR}" && touch "${PROFILE_FILE}"; then
      if ! grep -Fq "${PATH_EXPORT}" "${PROFILE_FILE}"; then
        if ! {
          echo ""
          echo "# Added by RobotX go installer"
          echo "${PATH_EXPORT}"
        } >> "${PROFILE_FILE}"; then
          echo "Warning: failed to write PATH to ${PROFILE_FILE}" >&2
        fi
      fi

      echo "Updated PATH in ${PROFILE_FILE}."
      echo "Run: source ${PROFILE_FILE}"
    else
      echo "Warning: cannot update ${PROFILE_FILE}" >&2
      echo "Add this line to your shell profile:" >&2
      echo "  ${PATH_EXPORT}" >&2
    fi
  else
    echo "Warning: ${INSTALL_DIR} is not on PATH" >&2
    echo "Add this line to your shell profile:" >&2
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\"" >&2
  fi
fi

if [[ -x "${INSTALL_DIR}/robotx" ]]; then
  echo "Installed robotx to ${INSTALL_DIR}/robotx"
  "${INSTALL_DIR}/robotx" --version || true
else
  echo "install failed: ${INSTALL_DIR}/robotx not found" >&2
  exit 1
fi
