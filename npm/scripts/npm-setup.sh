#!/usr/bin/env bash
# One-time npm setup for goblox releases (docs/goblox/RELEASING.md). Run it logged in to npm as the owner
# of goblox and @typetorch (npm whoami). Every publish and trust call asks for 2FA in the browser.
#
# 1. Publishes a 0.0.0 placeholder of each @typetorch/goblox-<os>-<arch> package that doesn't exist yet
#    (npm only accepts a trusted publisher on an existing package).
# 2. Makes typetorch/goblox-ts's .github/workflows/release.yml the trusted publisher of all eight packages.
#
# Safe to re-run: existing packages are not republished, and existing trust entries are listed, not added.
set -euo pipefail

REPO=typetorch/goblox-ts
WORKFLOW=release.yml
PLATFORMS=(win32-x64 win32-arm64 linux-x64 linux-arm64 darwin-x64 darwin-arm64)
PACKAGES=(goblox @typetorch/goblox)

echo "npm user: $(npm whoami)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

for platform in "${PLATFORMS[@]}"; do
	name="@typetorch/goblox-$platform"
	PACKAGES+=("$name")
	if npm view "$name" name > /dev/null 2>&1; then
		echo "$name exists"
		continue
	fi
	dir="$tmp/$platform"
	mkdir -p "$dir"
	cat > "$dir/package.json" <<EOF
{
	"name": "$name",
	"version": "0.0.0",
	"description": "Placeholder for the $platform binary of goblox. Real releases come from CI.",
	"repository": { "type": "git", "url": "git+https://github.com/$REPO.git" },
	"license": "Apache-2.0"
}
EOF
	printf '# %s\n\nPlaceholder, real releases come from CI (%s, `.github/workflows/%s`).\nInstall [goblox](https://www.npmjs.com/package/goblox) instead.\n' \
		"$name" "$REPO" "$WORKFLOW" > "$dir/README.md"
	echo "publishing the $name placeholder"
	(cd "$dir" && npm publish --access public)
done

for name in "${PACKAGES[@]}"; do
	existing="$(npm trust list "$name" 2>/dev/null || true)"
	if grep -q "$REPO" <<< "$existing"; then
		echo "$name already trusts $REPO"
		continue
	fi
	echo "trusting $REPO $WORKFLOW for $name"
	npm trust github "$name" --repo "$REPO" --file "$WORKFLOW" --allow-publish --yes
done

echo "done. Check with: npm trust list <package>"
