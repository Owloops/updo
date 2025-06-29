#!/usr/bin/env bash

set -euo pipefail

readonly BLUE='\033[0;34m'
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly RESET='\033[0m'

readonly TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT INT TERM

log() {
    local level=$1
    shift
    local color
    case "$level" in
        INFO) color="$GREEN" ;;
        WARN) color="$YELLOW" ;;
        ERROR) color="$RED" ;;
        *) color="$BLUE" ;;
    esac
    echo -e "${color}[$(date +'%Y-%m-%dT%H:%M:%S')] ${level}: $*${RESET}" >&2
}

ensure_command() {
    command -v "$1" >/dev/null 2>&1 || { log ERROR "Required command not found: $1"; exit 1; }
}

print_banner() {
    echo -e "${BLUE}"
    echo "██╗░░░██╗██████╗░██████╗░░█████╗░"
    echo "██║░░░██║██╔══██╗██╔══██╗██╔══██╗"
    echo "██║░░░██║██████╔╝██║░░██║██║░░██║"
    echo "██║░░░██║██╔═══╝░██║░░██║██║░░██║"
    echo "╚██████╔╝██║░░░░░██████╔╝╚█████╔╝"
    echo "░╚═════╝░╚═╝░░░░░╚═════╝░░╚════╝░"
    echo -e "${RESET}"
    echo "Website Monitoring Tool - Installer"
    echo
}

detect_platform() {
    OS="$(uname -s)"
    case "${OS}" in
        Linux*)     SYSTEM=Linux;;
        Darwin*)    SYSTEM=Darwin;;
        MINGW*)     SYSTEM=Windows;;
        MSYS*)      SYSTEM=Windows;;
        *)          log ERROR "Unsupported operating system: ${OS}"; exit 2;;
    esac

    ARCH="$(uname -m)"
    case "${ARCH}" in
        x86_64*)    ARCH=x86_64;;
        arm64*)     ARCH=arm64;;
        aarch64*)   ARCH=arm64;;
        i386*)      ARCH=i386;;
        i686*)      ARCH=i386;;
        *)          log ERROR "Unsupported architecture: ${ARCH}"; exit 2;;
    esac

    log INFO "Detected system: ${SYSTEM}_${ARCH}"
}

get_latest_version() {
    log INFO "Fetching latest release information"
    LATEST_RELEASE_URL="https://api.github.com/repos/Owloops/updo/releases/latest"
    RELEASE_DATA=$(curl -s -H "Accept: application/vnd.github.v3+json" "${LATEST_RELEASE_URL}")

    VERSION=$(echo "${RELEASE_DATA}" | grep -o '"tag_name":[^,}]*' | sed 's/.*"tag_name"[": ]*//;s/[",]//g')

    if [[ -z "${VERSION}" ]]; then
        log ERROR "Failed to get latest version. Check your internet connection."
        exit 1
    fi

    log INFO "Latest version: ${VERSION}"
}

get_download_url() {
    if [[ "${SYSTEM}" = "Windows" ]]; then
        ASSET_PATTERN="${SYSTEM}_${ARCH}.zip"
        ensure_command unzip
    else
        ASSET_PATTERN="${SYSTEM}_${ARCH}.tar.gz"
        ensure_command tar
    fi

    DOWNLOAD_URL=$(echo "${RELEASE_DATA}" | grep -o "\"browser_download_url\": \"[^\"]*${ASSET_PATTERN}\"" | grep -o "https://[^\"]*")

    if [[ -z "${DOWNLOAD_URL}" ]]; then
        log ERROR "Could not find download URL for ${SYSTEM}_${ARCH}"
        echo "Available assets:" >&2
        echo "${RELEASE_DATA}" | grep -o '"name": "[^"]*"' | grep -o '[^"]*$' | grep -v '^name$'
        exit 1
    fi
    
    log INFO "Found download URL: ${DOWNLOAD_URL}"
}

download_and_extract() {
    ARCHIVE_NAME=$(basename "${DOWNLOAD_URL}")
    ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_NAME}"
    
    log INFO "Downloading from ${DOWNLOAD_URL}"
    curl -SL -o "${ARCHIVE_PATH}" "${DOWNLOAD_URL}"
    
    log INFO "Extracting archive"
    if [[ "${ARCHIVE_NAME}" == *.zip ]]; then
        unzip -q "${ARCHIVE_PATH}" -d "${TMP_DIR}"
    else
        tar -xzf "${ARCHIVE_PATH}" -C "${TMP_DIR}"
    fi
}

install_binary() {
    INSTALL_DIR="/usr/local/bin"
    if [[ ! -d "${INSTALL_DIR}" ]] || [[ ! -w "${INSTALL_DIR}" ]]; then
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "${INSTALL_DIR}"
        
        if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
            log INFO "Adding ${INSTALL_DIR} to your PATH"
            
            SHELL_CONFIG=""
            if [[ -f "${HOME}/.zshrc" ]] && [[ "${SHELL}" == */zsh ]]; then
                SHELL_CONFIG="${HOME}/.zshrc"
            elif [[ -f "${HOME}/.bashrc" ]]; then
                SHELL_CONFIG="${HOME}/.bashrc"
            fi
            
            if [[ -n "${SHELL_CONFIG}" ]]; then
                echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "${SHELL_CONFIG}"
                log INFO "Added to ${SHELL_CONFIG}. Please restart your terminal or run 'source ${SHELL_CONFIG}'"
            else
                log WARN "Please add ${INSTALL_DIR} to your PATH manually."
            fi
        fi
    fi

    BINARY_PATH=$(find "${TMP_DIR}" -name "updo" -type f)
    if [[ -z "${BINARY_PATH}" ]]; then
        log ERROR "Failed to find the updo binary in the extracted archive"
        exit 1
    fi

    cp "${BINARY_PATH}" "${INSTALL_DIR}/updo"
    chmod +x "${INSTALL_DIR}/updo"

    if [[ "${SYSTEM}" = "Darwin" ]]; then
        log INFO "Removing macOS quarantine attribute"
        xattr -d com.apple.quarantine "${INSTALL_DIR}/updo" 2>/dev/null || true
    fi

    log INFO "updo ${VERSION} has been successfully installed to ${INSTALL_DIR}/updo"
}

verify_installation() {
    if command -v "${INSTALL_DIR}/updo" &>/dev/null; then
        log INFO "Installed version:"
        "${INSTALL_DIR}/updo" --version
    fi

    echo
    echo "You can now run updo from the command line."
    echo
    echo -e "${BLUE}Get started:${RESET}"
    echo "  updo --help                  # Show help and usage information"
    echo "  updo --version               # Show version information"
    echo
    echo -e "${BLUE}Enable shell completions:${RESET}"
    echo "  # For bash:"
    echo "  source <(updo completion bash)"
    echo
    echo "  # For zsh:"
    echo "  source <(updo completion zsh)"
    echo
    echo "  # For fish:"
    echo "  updo completion fish | source"
    echo
    echo -e "For full documentation, visit: ${GREEN}https://github.com/Owloops/updo#usage${RESET}"

    log INFO "Thank you for installing updo!"
}

main() {
    ensure_command curl
    ensure_command grep
    ensure_command sed
    
    print_banner
    detect_platform
    get_latest_version
    get_download_url
    download_and_extract
    install_binary
    verify_installation
}

main "$@"