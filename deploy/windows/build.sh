#!/usr/bin/env bash
# Builds vikunja.exe for Windows (amd64) into deploy/windows/out/ together with deploy.ps1.
# Needs Node + pnpm and Docker. Run from Git Bash anywhere inside the repo, on the branch you deploy (prod).
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

out="deploy/windows/out"
version=$(git describe --tags --always --abbrev=10 --dirty | sed 's/-g/-/')
go_version=$(awk '/^go /{print $2}' go.mod | cut -d. -f1,2)

(cd frontend && PUPPETEER_SKIP_DOWNLOAD=1 pnpm install --frozen-lockfile && pnpm build)

rm -rf "$out"
mkdir -p "$out"
cp deploy/windows/deploy.ps1 "$out/"

# The SQLite driver needs cgo, so cross-compile with MinGW inside a Go container.
MSYS_NO_PATHCONV=1 docker run --rm \
	-v "$(pwd -W):/src" \
	-v vikunja-gomod:/go/pkg/mod \
	-v vikunja-gocache:/root/.cache/go-build \
	-w /src \
	-e GOOS=windows -e GOARCH=amd64 -e CGO_ENABLED=1 -e CC=x86_64-w64-mingw32-gcc \
	"golang:${go_version}-bookworm" \
	bash -c "apt-get update -qq >/dev/null \
		&& apt-get install -y -qq gcc-mingw-w64-x86-64 >/dev/null 2>&1 \
		&& go build -buildvcs=false -tags osusergo \
			-ldflags '-s -w -X \"code.vikunja.io/api/pkg/version.Version=${version}\" -X \"main.Tags=osusergo\"' \
			-o ${out}/vikunja.exe ."

echo "Built ${out}/vikunja.exe (${version})."
echo "Copy ${out} to the server and run there, as Administrator: powershell -ExecutionPolicy Bypass -File .\\deploy.ps1"
