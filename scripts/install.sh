#!/usr/bin/env bash
set -euo pipefail

REPO="${ROBOTX_REPO:-haibingtown/robotx_cli}"
REQUESTED_VERSION="${ROBOTX_VERSION:-latest}"
INSTALL_DIR="${ROBOTX_INSTALL_DIR:-${HOME}/.local/bin}"
AUTO_PATH="${ROBOTX_AUTO_PATH:-1}"
CURL_CONNECT_TIMEOUT="${ROBOTX_CONNECT_TIMEOUT:-10}"
CURL_MAX_TIME="${ROBOTX_MAX_TIME:-300}"
CURL_ARGS=(-fsSL --connect-timeout "${CURL_CONNECT_TIMEOUT}" --max-time "${CURL_MAX_TIME}")

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

if ! command -v tar >/dev/null 2>&1; then
  echo "tar is required" >&2
  exit 1
fi

OS_RAW="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_RAW="$(uname -m | tr '[:upper:]' '[:lower:]')"

case "${OS_RAW}" in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  *)
    echo "unsupported OS: ${OS_RAW}" >&2
    exit 1
    ;;
esac

case "${ARCH_RAW}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "unsupported architecture: ${ARCH_RAW}" >&2
    exit 1
    ;;
esac

resolve_tag() {
  if [[ "${REQUESTED_VERSION}" == "latest" ]]; then
    local tag
    echo "Resolving latest release tag from ${REPO}..." >&2
    tag="$(
      curl "${CURL_ARGS[@]}" "https://api.github.com/repos/${REPO}/releases/latest" \
        | sed -nE 's/^[[:space:]]*"tag_name":[[:space:]]*"([^"]+)".*/\1/p' \
        | tail -n1
    )"
    if [[ -z "${tag}" ]]; then
      echo "failed to resolve latest release tag from ${REPO}" >&2
      exit 1
    fi
    echo "${tag}"
    return
  fi

  if [[ "${REQUESTED_VERSION}" == v* ]]; then
    echo "${REQUESTED_VERSION}"
  else
    echo "v${REQUESTED_VERSION}"
  fi
}

TAG="$(resolve_tag)"
VERSION="${TAG#v}"
ARCHIVE_NAME="robotx_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUMS_NAME="checksums.txt"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE_NAME}"
CHECKSUMS_URL="${BASE_URL}/${CHECKSUMS_NAME}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

echo "Downloading ${ARCHIVE_NAME} from ${TAG}..."
curl "${CURL_ARGS[@]}" "${ARCHIVE_URL}" -o "${TMP_DIR}/${ARCHIVE_NAME}"
echo "Downloading ${CHECKSUMS_NAME} from ${TAG}..."
curl "${CURL_ARGS[@]}" "${CHECKSUMS_URL}" -o "${TMP_DIR}/${CHECKSUMS_NAME}"

EXPECTED_SUM="$(awk -v file="${ARCHIVE_NAME}" '$2 == file {print $1}' "${TMP_DIR}/${CHECKSUMS_NAME}")"
if [[ -z "${EXPECTED_SUM}" ]]; then
  echo "missing checksum for ${ARCHIVE_NAME}" >&2
  exit 1
fi

echo "Verifying checksum..."
if command -v shasum >/dev/null 2>&1; then
  ACTUAL_SUM="$(shasum -a 256 "${TMP_DIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  ACTUAL_SUM="$(sha256sum "${TMP_DIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
else
  echo "shasum or sha256sum is required" >&2
  exit 1
fi

if [[ "${EXPECTED_SUM}" != "${ACTUAL_SUM}" ]]; then
  echo "checksum mismatch for ${ARCHIVE_NAME}" >&2
  exit 1
fi

mkdir -p "${INSTALL_DIR}"

echo "Installing robotx to ${INSTALL_DIR}..."
tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}"
install -m 0755 "${TMP_DIR}/robotx" "${INSTALL_DIR}/robotx"

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

echo "Installed robotx ${TAG} to ${INSTALL_DIR}/robotx"
if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  if [[ "${AUTO_PATH}" == "1" ]]; then
    PROFILE_FILE="$(detect_profile_file)"
    PATH_EXPORT="export PATH=\"${INSTALL_DIR}:\$PATH\""
    PROFILE_DIR="$(dirname "${PROFILE_FILE}")"

    if mkdir -p "${PROFILE_DIR}" && touch "${PROFILE_FILE}"; then
      if ! grep -Fq "${PATH_EXPORT}" "${PROFILE_FILE}"; then
        if ! {
          echo ""
          echo "# Added by RobotX installer"
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
