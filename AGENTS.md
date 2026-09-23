# nexus-util — instructions for coding agents

`nexus-util` is a single-binary Go CLI for managing assets in a Nexus OSS **Raw**
repository: push, pull, delete, list, diff, sync between two Nexus instances, plus
repository and blob-store inspection. It is a client only — no database, no server
component, no background workers, no generated code, no migrations.

Read this file before editing. There is no nested `AGENTS.md` in this repository.

## 1. Repository map

| Path | Package | Purpose |
| --- | --- | --- |
| `main.go` | `main` | The only `package main`. Registers every cobra command and all global flags (`setupCommands()`). |
| `cmd/asset/` | `asset` | `asset push\|pull\|delete\|list\|diff` — file/directory transfer and comparison. |
| `cmd/blob/` | `blob` | `blob create\|list\|show` — blob stores. |
| `cmd/repo/` | `repo` | `repo ls` — list repositories. |
| `cmd/sync/` | `sync` | `sync` — copy a repository to another repository. |
| `cmd/init/` | `initcmd` | `init` — writes the config file. The **directory** is `cmd/init`, the **package** is `initcmd`. |
| `config/` | `config` | YAML config load/save/validate and viper wiring. |
| `nexus/` | `nexus` | The only HTTP client: every Nexus REST call, streaming upload/download, hashing. |
| `scripts/install-tools.sh` | — | Installs pinned dev tools into `.artifacts/bin` (checksum-verified). |
| `.github/workflows/build.yml` | — | CI: verify job, 9-platform build matrix, tag-only release. |

Notable absences — do not invent them: there is no `internal/`, `pkg/`, `api/`,
`migrations/`, `deploy/`, `testdata/`, `vendor/`, `go.work`, protobuf/OpenAPI/GraphQL
schema, Docker Compose, Kubernetes or Terraform code.

### Command surface (verified against `nexus-util --help`)

```
nexus-util asset {push,pull,delete,list,diff}   # -r/--repository is required
nexus-util blob  {create,list,show}
nexus-util repo  ls
nexus-util sync
nexus-util init
nexus-util completion {bash,zsh,fish,powershell}
```

Global flags on every command: `-a/--address`, `-u/--user`, `-p/--password`,
`-c/--config`, `-q/--quiet`, `--dry`, `--insecure`.

Two traps that break naive edits and documentation:

- Subcommands are **nested**. `nexus-util push` fails with `unknown command "push"`;
  the real form is `nexus-util asset push`. Older docs and commit messages use the
  flat form.
- `-r/--repository` is defined on `asset` only, **not** globally, even though it
  appears in most examples.

## 2. Prerequisites

- Go, version from `go.mod`. Do not bump the `go` directive as a
  drive-by change.
- `make` for the normalized commands. All targets work from the repository root.
- No C toolchain is needed for `make build`/`make test` (`CGO_ENABLED=0` is exported
  by the Makefile). `make test-race` overrides it to `CGO_ENABLED=1` and therefore
  does need a working cgo toolchain.
- Docker is optional and only used by `make run-container`, which pulls from a
  private registry (`nexus.redkit-lab.work:8084` in `Dockerfile`). Do not run it
  without registry access; it is never part of verification.

Note: `go.mod` declares `go 1.21` while the `Dockerfile` installs Go 1.25.4. CI now
follows `go-version-file: go.mod`, so treat the `go.mod` directive as the source of
truth for the language version. `make vulncheck` needs Go >= 1.22 for govulncheck
itself; on a strict 1.21 toolchain it relies on Go's default `GOTOOLCHAIN=auto`
switching. With `GOTOOLCHAIN=local` it will not run.

Bootstrap (network required, writes only into git-ignored `.artifacts/`):

```bash
make bootstrap   # go mod download + install pinned golangci-lint
```

`scripts/install-tools.sh` downloads the pinned golangci-lint release, verifies its
sha256 against the release checksums file, and installs it into `.artifacts/bin`.
It is idempotent and installs nothing outside that directory. golangci-lint is
pinned to **v1** deliberately: `.golangci.yml` uses the v1 configuration schema and
will not load in golangci-lint v2.

## 3. Commands

Everything below is a real, copy-pasteable target. Run from the repository root.

| Command | What it does |
| --- | --- |
| `make help` | List all targets and the current variable values. |
| `make bootstrap` | Dependencies + pinned local tools. |
| `make fmt` | Format in place: `gofmt -s`, `goimports -local nexus-util`, whitespace. |
| `make fmt-check` | Read-only. Fail if anything is not formatted. |
| `make lint` | golangci-lint on issues **introduced since `origin/main`** (see §4). |
| `make lint-full` | golangci-lint over the whole repository — reports the pre-existing debt. |
| `make vet` | `go vet ./...`. |
| `make test` | `go test ./...`. |
| `make test-race` | `CGO_ENABLED=1 go test -race ./...`. |
| `make test-cover` | Writes `.artifacts/coverage.out` and prints the total. |
| `make build` | Builds the current platform into `bin/nexus-util`. |
| `make generate` | Runs `go generate ./...`, or reports that no generators exist. |
| `make generate-check` | Fails if generation left uncommitted changes. |
| `make vulncheck` | `govulncheck` (pinned). Needs network access to `vuln.go.dev`. |
| `make verify` | **The gate**: `fmt-check vet lint test build generate-check`. |
| `make clean` | Removes `.artifacts/`, `bin/`, `release/`. |

Narrow your scope instead of running the whole matrix:

```bash
go test ./nexus/...                 # one package
go test -run TestUploadFile ./nexus # one test
make test GO_TEST_FLAGS='-count=1 -v'
make lint LINT_NEW_FROM=origin/main
```

There is no separate integration/e2e suite: the tests are hermetic and need no
network, no Nexus and no credentials.

## 4. Working order

1. Read this file, then the package you are about to change.
2. Locate the smallest affected package. Most behavior changes belong in `nexus/`
   (protocol/transfer) or in a single `cmd/*` package (flag/UX wiring).
3. Make the minimal change; do not reformat or refactor unrelated files.
4. Add or update tests when behavior changes (§6).
5. Run `make fmt`, then the narrow test for the package you touched, then
   `make verify` before declaring the task done.
6. Report only commands you actually ran, with their real results.

**Lint is incremental by design.** `make lint` defaults to
`--new-from-rev=$(git merge-base HEAD origin/main)` so it reports issues in the
current branch/working tree, not the ~48 known pre-existing ones. Do not "fix" the
legacy list as part of an unrelated task, and do not add new `//nolint` directives
to silence a finding you introduced — fix it, or leave an inline, explained
exclusion.

## 5. Go conventions in this repository

- `gofmt -s` is mandatory. Import groups are: standard library, blank line,
  `nexus-util/...`, blank line, third party (see `main.go`, `cmd/asset/list.go`).
  `make fmt` produces exactly this.
- Keep dependencies minimal. Direct dependencies are only `spf13/cobra`,
  `spf13/viper` and `gopkg.in/yaml.v3`. Justify any new one; never add a dependency
  for something the standard library does.
- Do not change a public API or a configuration contract silently. The config
  contract is the flat YAML keys `nexusAddress`, `user`, `password` written by
  `config.SaveConfig`; `init` output is the source of truth for that format.
- Wrap errors with context and `%w`; do not swallow them. `_ = cmd.Flags().GetString(...)`
  is the pre-existing convention in `cmd/*` for flags that are known to be
  registered — keep it consistent locally rather than mass-refactoring.
- Never edit `go.sum`/`go.mod` by hand; use `go get` / `make tidy` and review the diff.

### Invariants that are easy to break

- **`http.Client.Timeout` is 0 on purpose.** `nexus.NewNexusClient` sets
  `Timeout: 0` and relies on per-request contexts (`downloadTimeout`,
  `uploadTimeout`) plus `responseHeaderTimeout`. A client-wide timeout truncates
  multi-GB transfers with an unexpected EOF. `nexus_test.go` asserts these values —
  do not "fix" the missing timeout.
- **Downloads stream to disk.** `DownloadFileByUrl` uses `io.Copy` on purpose;
  buffering a whole asset in memory breaks files over ~4 GiB.
- **Config is flat.** `viper.Unmarshal` into `config.Config` does not map a nested
  `nexus: {address: ...}` block. Only `nexusAddress`/`user`/`password` work.
- The `--insecure` flag sets `InsecureSkipVerify` (flagged by gosec). That is
  intended and user-requested; do not "harden" it away without a product decision.

## 6. Testing

- Put table-driven tests where they add scenario coverage; keep names
  `TestThing(t *testing.T)`.
- Test helpers call `t.Helper()`.
- Tests must not depend on wall-clock time, the network or real credentials.
  `nexus` tests use `httptest.NewServer`; use `t.TempDir()` for files.
- `config` tests manipulate the global `viper` singleton, so they call
  `viper.Reset()` first. Keep that pattern when adding tests there.
- `cmd/*` packages currently have no tests. Adding a first test there is welcome
  but keep it hermetic (flag parsing and pure helpers, not live Nexus calls).
- Coverage today (repo-wide, `make test-cover`): ~9%. `config` ~61%, `nexus` ~19%,
  `cmd/*` and `main` 0%. Raising coverage in `nexus/` is the highest-value work.

## 7. Security and side effects

Safe to run freely: `make help`, `fmt-check`, `lint`, `lint-full`, `vet`, `test`,
`test-race`, `test-cover`, `build`, `generate`, `generate-check`, `verify`, `clean`.

Requires explicit human approval before running:

- Anything that contacts a real Nexus: `asset push`, `asset pull`, `asset delete`,
  `asset sync`, `asset diff`, `blob`, `repo ls`, `init` against real credentials.
  These mutate remote state (delete/sync/push) or write local files (pull).
- `make run-container` (Docker + private registry), `make release`, `make build-all`
  if disk or CI capacity matters.
- `git commit`, `git push`, tag creation, PR/release creation. Never rewrite history.

Never do any of this without an explicit request: apply migrations (none exist),
`kubectl`/`terraform`/`helm` (not used here), publish releases or tags, push
branches, or contact production APIs.

Secrets: the repository contains no real credentials — `test-config.yaml` and the
README use placeholders such as `testpass`. `config.SaveConfig` writes the config
file with mode `0600`. Do not add real tokens, passwords or keys to any file, do
not paste them into commands you report, and never add a path that prints
`cfg.GetPassword()`. If you find a real secret committed anywhere, stop and report
it to the human instead of committing, deleting, or rewriting history; rotation is
the human's call.

`--dry` is a real guard at the HTTP mutation layer (`DeleteFile`, `DownloadFileByUrl`,
`UploadFile`, `UploadFromBuffer`, `TransferFile` all honor `DryRun`), but read-only
`HEAD`/search requests still reach the server, and `asset sync --dry` still creates a
temporary local file. Dry-run is not a substitute for pointing at a scratch
repository.

## 8. Definition of done

- `make fmt-check` passes (run `make fmt` first if it does not).
- `make lint` passes for the change (no new findings, no new blanket `//nolint`).
- Affected tests were added or updated, and `make test` passes.
- `make verify` passes end to end.
- If user- or operator-facing behavior changed, `README.md` was updated to match
  real command output.
- `git diff --check` is clean, and `git status --short` contains only intended files
  (no `bin/`, no `.artifacts/`, no stray probe files).
- Generated artifacts are committed and reproduce byte-for-byte (there are no
  generators today; `make generate-check` says so explicitly).
