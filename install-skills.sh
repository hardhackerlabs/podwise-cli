#!/bin/sh
set -e

REPO="hardhackerlabs/podwise-cli"
FILE_NAME="podwise-skills.tar.gz"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo "==> Installing Podwise skills from $REPO..."

# Fetch latest version if not specified
if [ -z "$VERSION" ]; then
    echo "==> Fetching latest release version..."
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "${RED}Error: Could not determine the latest version from GitHub.${NC}"
        exit 1
    fi
fi

echo "==> Version: $VERSION"

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$FILE_NAME"

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "==> Downloading $FILE_NAME..."
curl -sL -o "$TMP_DIR/$FILE_NAME" "$DOWNLOAD_URL"

echo "==> Extracting to current directory..."
tar xzf "$TMP_DIR/$FILE_NAME" -C .

INSTALL_DIR="$(pwd)/podwise-skills"

echo ""
echo "${GREEN}==> Skills downloaded to: ${INSTALL_DIR}${NC}"
echo ""
echo "==> Available skills:"
if [ -d "$INSTALL_DIR" ]; then
    for skill_dir in "$INSTALL_DIR"/*/; do
        if [ -d "$skill_dir" ]; then
            skill_name=$(basename "$skill_dir")
            printf "    - %s\n" "$skill_name"
        fi
    done
else
    echo "    (no skills directory found, please check the extracted content)"
fi

echo ""
echo "==> Next steps: install the skills into your agent environment as needed."
echo ""
echo "    OpenAI Codex:"
echo "      cp -r ${INSTALL_DIR}/* ~/.agents/skills/"
echo ""
echo "    Gemini CLI:"
echo "      cp -r ${INSTALL_DIR}/* ~/.gemini/skills/"
echo ""
echo "    Cursor:"
echo "      cp -r ${INSTALL_DIR}/* ~/.cursor/skills-cursor/"
echo ""
echo "    Claude Code:"
echo "      cp -r ${INSTALL_DIR}/* ~/.claude/skills/"
echo ""
