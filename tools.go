//go:build tools

// Package tools tracks development tooling as module dependencies so they
// land in go.sum and install at pinned versions via `make dev-setup`.
//
// Production binaries never compile these imports — the `tools` build tag
// excludes this file from any normal build.
package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/rhysd/actionlint/cmd/actionlint"
	_ "github.com/zricethezav/gitleaks/v8" // gitleaks module path; the repo moved to the gitleaks org but the module path did not
	_ "golang.org/x/vuln/cmd/govulncheck"
)
