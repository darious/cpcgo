#!/usr/bin/env sh
set -eu

export GOMODCACHE="${GOMODCACHE:-/tmp/cpcgo-go-mod}"
export GOCACHE="${GOCACHE:-/tmp/cpcgo-go-build}"

echo "==> gofmt"
unformatted="$(gofmt -l $(find cmd internal -name '*.go' -type f))"
if [ -n "$unformatted" ]; then
	echo "Go files need formatting:"
	echo "$unformatted"
	exit 1
fi

echo "==> go test"
go test ./...

echo "==> go vet"
go vet ./...

echo "==> live UI compile"
go test -c -tags liveui -o /tmp/cpcgo-ui.test ./cmd/cpcgo-ui

echo "==> CLI ROM smoke test"
go run ./cmd/cpcgo --rom cpc6128.rom --amsdos amsdos.rom >/dev/null

echo "all tests passed"
