# Releasing goblox

`.github/workflows/release.yml` builds goblox for six platforms, publishes it to npm and creates a GitHub
Release. Pushing a `v*` tag starts it.

## Cutting a release

```sh
git checkout goblox && git pull
git tag v0.1.0            # or v0.1.0-beta.1 for a prerelease
git push origin v0.1.0
```

- **Version.** It comes from the tag: `v1.2.3` gives `1.2.3` for every package and for `goblox --version`
  (`-X main.version`). The `version = "0.1.0"` in `cmd/goblox/main.go` is only the default for local builds.
- **npm dist-tag.** A version with a prerelease part (`1.2.3-beta.1`) goes to `next`. A plain version goes to
  `latest`. The GitHub Release is marked as a prerelease in the same way.

### What the workflow does

1. **Version.** It reads the version from the tag and rejects anything that isn't semver.
2. **Build** (a matrix on ubuntu).
   - It checks out goblox-ts and `typetorch/transformer` side by side, at `TRANSFORMER_REF` in the workflow.
   - It runs `go work init ./goblox-ts ./transformer/go`.
   - It cross-compiles with `CGO_ENABLED=0 go build -tags typetorch -trimpath -ldflags "-s -w -X main.version=<version>"`
     (CLI.md, "Linking @typetorch/transformer into the release binary", option A).
   - It fails if the binary wasn't built with `-tags typetorch`.
   - Targets: `linux-x64`, `linux-arm64`, `darwin-x64`, `darwin-arm64`, `win32-x64`, `win32-arm64`.
3. **Package.**
   - `npm/scripts/assemble.mjs` assembles the packages, and `npm pack` packs each one.
   - The raw binaries are renamed to `goblox-<version>-<platform>[.exe]`, and `SHA256SUMS` is written for them.
4. **Smoke test.**
   - It installs the linux-x64, `goblox` and `@typetorch/goblox` tarballs into an empty project.
   - It checks that `npx goblox --version` prints the version.
   - It compiles `npm/smoke` with the installed bin, using `npm/smoke/check.sh`. The fixture's tsconfig
     lists `@typetorch/transformer`, so the build fails if the Go port isn't linked.
5. **Publish.**
   - Order: the six `@typetorch/goblox-<os>-<arch>` packages, then `goblox`, then `@typetorch/goblox`.
   - Command: `npm publish --provenance --access public --tag <latest|next>`, with trusted publishing (OIDC),
     so no npm token is involved.
   - A version that is already on npm is skipped, so re-running a half-failed release is safe.
6. **GitHub Release.** It creates the release for the tag, with the binaries and `SHA256SUMS` attached.

### Dry runs and retries

The Actions tab has a **Run workflow** button with three inputs:

- `dry_run` (the default): runs everything up to `npm publish --dry-run`, and creates no GitHub Release.
  - `version`: the version a dry run from a branch builds with (default `0.0.0-dryrun.<run number>`).
  - `transformer_ref`: links a different transformer commit.
- On a `v*` tag with `dry_run` unticked: publishes that tag again. Use this to retry a failed release; the
  versions already published are skipped.

GitHub only shows workflow_dispatch workflows that are on the repository's **default branch**. That branch is
`main` here (the upstream mirror), so the button and `gh workflow run release.yml --ref goblox` stay
unavailable until one of two things happens:

- the default branch becomes `goblox` (Settings → General → Default branch), or
- `release.yml` is also committed to `main`.

Tag pushes work either way: they run the workflow file from the tagged commit.

### Bumping the transformer

`TRANSFORMER_REF` in `release.yml` pins the `typetorch/transformer` commit (its `go/` folder) that releases
link. After transformer changes, set it to the new commit, or to a tag once the Go port is tagged.

## One-time setup (before the first tag)

### 1. Make sure every npm package exists

npm only lets you configure a trusted publisher on a package that already exists. As of 2026-10-30:

| Package | Status |
| --- | --- |
| `goblox` | exists (0.0.0 placeholder, maintainer `phasenull`) |
| `@typetorch/goblox` | exists (0.0.0 placeholder, maintainer `phasenull`) |
| `@typetorch/goblox-{win32,linux,darwin}-{x64,arm64}` | **do not exist**: publish a placeholder once |

Publish the six placeholders from a machine logged in to npm as `phasenull` (`npm whoami`). Each one is a
`0.0.0-placeholder` with no binary:

```sh
tmp=$(mktemp -d) && cd "$tmp"
for p in win32-x64 win32-arm64 linux-x64 linux-arm64 darwin-x64 darwin-arm64; do
  mkdir "$p" && cat > "$p/package.json" <<EOF
{ "name": "@typetorch/goblox-$p", "version": "0.0.0-placeholder",
  "description": "goblox $p binary. Placeholder; install goblox.",
  "repository": { "type": "git", "url": "git+https://github.com/typetorch/goblox-ts.git" },
  "license": "Apache-2.0" }
EOF
  (cd "$p" && npm publish --access public --tag placeholder)
done
```

`goblox` pins exact versions of its platform packages, so the placeholders are never installed. The first
plain release takes over `latest`.

### 2. Configure the trusted publisher on each of the 8 packages

On npmjs.com, open each package's **Settings** tab. Under **Trusted Publisher**, choose **GitHub Actions** and
enter:

| Field | Value |
| --- | --- |
| Organization or user | `typetorch` |
| Repository | `goblox-ts` |
| Workflow filename | `release.yml` |
| Environment | *(leave empty)* |

Do this for `goblox`, `@typetorch/goblox`, and the six `@typetorch/goblox-<os>-<arch>` packages.

Optionally, set **Publishing access** to "Require two-factor authentication and disallow tokens" on the same
page. Trusted publishing keeps working, and no leaked token can publish.

### 3. Secrets

None. npm authenticates the workflow through OIDC (`id-token: write`). `typetorch/transformer` is public, so
the checkout needs no token.

If the transformer repo is ever made private, do the following:

- Create a fine-grained token with **Contents: read** on `typetorch/transformer`.
- Store it as the repository secret **`TRANSFORMER_READ_TOKEN`** in typetorch/goblox-ts.
- Add `token: ${{ secrets.TRANSFORMER_READ_TOKEN }}` to the transformer checkout step in `release.yml`.

### 4. Actions permissions

In typetorch/goblox-ts, go to Settings → Actions → General. Actions must be allowed. The workflow sets its
own permissions per job: `id-token: write` for publish, and `contents: write` for the GitHub Release.

## Building the packages locally

```sh
# in a folder that has goblox-ts/ and transformer/ (typetorch/transformer) side by side
go work init ./goblox-ts ./transformer/go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags typetorch -trimpath \
  -ldflags "-s -w -X main.version=0.1.0-local" -o bin/linux-x64/goblox ./goblox-ts/cmd/goblox
node goblox-ts/npm/scripts/assemble.mjs --version 0.1.0-local --binaries bin --out npm-out --allow-missing
(cd npm-out/goblox && npm pack)
```

`--allow-missing` skips the platforms you didn't build, and leaves them out of `goblox`'s
`optionalDependencies`.

To run a local binary through an installed `goblox` package, set `GOBLOX_BINARY=/path/to/goblox`.
