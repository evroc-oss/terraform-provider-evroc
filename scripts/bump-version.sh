#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2026 evroc
#
# Automated version bump script for evroc Terraform provider
# Usage: ./scripts/bump-version.sh <version>
#        ./scripts/bump-version.sh v1.1.0
#        ./scripts/bump-version.sh 1.1.0

set -euo pipefail

if [ -z "$1" ]; then
    echo "Error: Version number required"
    echo "Usage: ./scripts/bump-version.sh <version>"
    echo "Example: ./scripts/bump-version.sh v1.1.0"
    exit 1
fi

INPUT_VERSION="$1"
CHANGELOG_FILE="CHANGELOG.md"

# Strip 'v' prefix if present
NEW_VERSION="${INPUT_VERSION#v}"

# Validate version format (semantic versioning)
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Invalid version format '$INPUT_VERSION'"
    echo "Version must be in format: [v]major.minor.patch (e.g., v1.1.0 or 1.1.0)"
    exit 1
fi

# Check if this version has already been released
ESCAPED_VERSION="${NEW_VERSION//./\\.}"
if [ -f "$CHANGELOG_FILE" ] && grep -qE "## \[$ESCAPED_VERSION\]" "$CHANGELOG_FILE"; then
    echo "WARNING: Version $NEW_VERSION already exists in $CHANGELOG_FILE"
    read -p "   Continue anyway? (yes/no): " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        echo "Aborted"
        exit 1
    fi
fi

# Get current version from latest git tag
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "unknown")

echo "Current version: $CURRENT_VERSION"
echo "New version: $NEW_VERSION"

echo ""

# Track updated files
UPDATED_FILES=()

# 1. Update CHANGELOG.md
if [ -f "$CHANGELOG_FILE" ]; then
    echo "Updating $CHANGELOG_FILE..."
    TODAY=$(date +%Y-%m-%d)
    sed -i "s/## \[Unreleased\]/## [Unreleased]\n\n## [$NEW_VERSION] - $TODAY/" "$CHANGELOG_FILE"

    # Add [Unreleased] link at the bottom if it doesn't exist
    if ! grep -q "\[Unreleased\]:" "$CHANGELOG_FILE"; then
        echo "" >> "$CHANGELOG_FILE"
        echo "[Unreleased]: https://github.com/evroc-oss/terraform-provider-evroc/compare/v$NEW_VERSION...HEAD" >> "$CHANGELOG_FILE"
    else
        # Update existing [Unreleased] link to compare from new version
        sed -i "s|\[Unreleased\]:.*|[Unreleased]: https://github.com/evroc-oss/terraform-provider-evroc/compare/v$NEW_VERSION...HEAD|" "$CHANGELOG_FILE"
    fi

    # Add version link at the bottom if it doesn't exist
    if ! grep -q "\[$NEW_VERSION\]:" "$CHANGELOG_FILE"; then
        echo "[$NEW_VERSION]: https://github.com/evroc-oss/terraform-provider-evroc/releases/tag/v$NEW_VERSION" >> "$CHANGELOG_FILE"
    fi

    UPDATED_FILES+=("$CHANGELOG_FILE")
fi

# 2. Update Makefile VERSION
if [ -f "Makefile" ]; then
    echo "Updating Makefile..."
    sed -i "s|^VERSION?=.*|VERSION?=$NEW_VERSION|" Makefile
    UPDATED_FILES+=("Makefile")
fi

# README.md, docs/VERIFICATION.md, docs/RELEASING.md, and examples use generic
# placeholders or no version constraints, so they don't need version-specific updates.

# Stage and commit (no tag — tag is created via sync script)
echo ""
echo "Staging changes..."
git add "${UPDATED_FILES[@]}"

echo "Creating commit..."
git commit -m "chore: bump version to v$NEW_VERSION"

echo ""
echo "Version bumped successfully!"
echo "   Old: v$CURRENT_VERSION"
echo "   New: v$NEW_VERSION"
echo ""
echo "Files updated:"
for f in "${UPDATED_FILES[@]}"; do
    echo "   - $f"
done
echo ""
echo "Next steps:"
echo "   git push origin $(git branch --show-current)"
echo "   ./scripts/sync-to-github.sh --main"
echo "   # Review and merge on GitHub, then:"
echo "   ./scripts/sync-to-github.sh --tag v$NEW_VERSION"
echo ""
