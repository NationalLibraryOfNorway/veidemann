# CLI Tools — Agent-Focused Overview

## Top 10 — Highest Agent Value

| # | Tool | Why |
|---|------|-----|
| 1 | **rg** | Primary code search: 5–10× faster than grep, respects `.gitignore` — fewer hits to filter = fewer turns |
| 2 | **jq** + **yq** | Trim JSON/YAML responses before they fill context — massive token savings on every API call |
| 3 | **gh** | Only practical path to GitHub data (PRs, issues, CI runs, review comments) without HTML scraping |
| 4 | **ast-grep** | Semantic search/refactor at AST level — eliminates entire classes of false text matches |
| 5 | **duckdb** | SQL against CSV/JSON/Parquet without import — replaces 20+ lines of Python/awk for ad-hoc analysis |
| 6 | **ruff** | Python lint + format in milliseconds — tight feedback loop, no iteration over style errors |
| 7 | **fd** | File search with short syntax and `.gitignore` respect — no `find . -name` syntax to remember |
| 8 | **semgrep** | Pattern-based code analysis across all languages in one run (`--config auto`) |
| 9 | **act** | Run GitHub Actions locally before push — saves minutes to hours per CI iteration |
| 10 | **gron** | Flattens JSON to greppable lines — one pipe instead of 3 jq attempts when structure is unknown |

### By type of benefit

**Saves most turns** — rg, fd, ast-grep, ruff, biome, shellcheck, actionlint, act, hurl, git-absorb
**Saves most tokens** — jq, yq, jc, gron, duckdb, super, miller, htmlq, fq, hexyl
**Provides data otherwise inaccessible** — gh, grpcurl, websocat, mitmdump, fq, trufflehog, exiftool,
skopeo
**Catches errors before execution** — ruff, biome, pyright, shellcheck, actionlint, hadolint, kubeconform, conftest, semgrep

---

## Contents

- [Search & Code Navigation](#search--code-navigation)
- [JSON, Data & Structured Output](#json-data--structured-output)
- [Git & Versioning](#git--versioning)
- [Text Manipulation](#text-manipulation)
- [Documents & Metadata](#documents--metadata)
- [HTTP, API & Networking](#http-api--networking)
- [Linting & Static Analysis](#linting--static-analysis)
- [Security Scanning](#security-scanning)
- [Kubernetes & Infrastructure](#kubernetes--infrastructure)
- [Build, Watch & Bench](#build-watch--bench)
- [Container Introspection](#container-introspection)
- [Secrets & Encryption](#secrets--encryption)
- [Runtimes (only what is actually used)](#runtimes-only-what-is-actually-used)

---

## Search & Code Navigation

| Tool | Agent benefit |
|------|--------------|
| **rg** (ripgrep) | Primary code search tool. Respects `.gitignore`, 5–10× faster than grep. Fewer false hits = fewer turns. |
| **rga** (ripgrep-all) | Extends `rg` to document formats (PDF, Office, ZIP, e-books, etc.). Critical when the bug lies in artifacts, not just source code. |
| **fd** | File search without writing `find . -name` syntax. Respects `.gitignore`. Shorter commands = fewer tokens. |
| **ast-grep** | Structural code search and refactor at AST level. Matches semantics, not text — eliminates entire classes of false hits. Irreplaceable for safe refactors. |

## JSON, Data & Structured Output

| Tool | Agent benefit |
|------|--------------|
| **jq** | Standard JSON processor. Indispensable for trimming API responses before they fill context. |
| **yq** | Same for YAML/TOML/XML. Critical for k8s manifests, CI config, Helm. |
| **jc** | Converts messy CLI output to JSON (`jc … | jq`). Dramatically fewer parsing errors and tokens than regex/awk on free text. |
| **gron** | Flattens JSON to `path = value` lines that are greppable. Saves iterations when structure is unknown — one pipe instead of 3 jq attempts. |
| **miller** (mlr) | Tabular transform (CSV/TSV/JSON) in one command. Replaces awk pipelines. |
| **htmlq** | CSS selectors against HTML — direct extraction without manually parsing markup. |
| **sqlite3** | Inspects SQLite databases directly. Many tools (atuin, browsers, apps) use SQLite. |
| **duckdb** | SQL against CSV/JSON/Parquet without import. Replaces 20+ lines of Python/awk for ad-hoc analysis. Massive turn savings. |
| **jo** | Builds JSON from args: `jo a=1 b=2`. Eliminates escape bugs in API calls and fixture generation. |
| **fq** (wader/fq) | `jq` syntax against binary formats (pcap, zip/jar, protobuf, SQLite wire, etc.). Only practical way to inspect binary files structurally — without it I have to guess. |
| **hexyl** | Readable hex dump with color coding for ASCII/control/null. Fallback when `fq` lacks a parser — gives structure to unknown binaries instantly. |

## Git & Versioning

| Tool | Agent benefit |
|------|--------------|
| **git** | The foundation. |
| **gh** | GitHub API from shell — PRs, issues, workflow runs, review comments. Only practical path to GitHub data without scraping HTML. |
| **difft** (difftastic) | Structural AST diff. Separates semantic changes from formatting — saves turns on "is this a real change?". |
| **git-absorb** | Auto-fixup changes into the correct previous commit based on blame. One command instead of manually finding the target commit during iterative refactor. |
| **scc** | Lines of code + complexity + COCOMO in one run. Quick codebase overview at single-command level. |

## Text Manipulation

| Tool | Agent benefit |
|------|--------------|
| **sd** | Find-and-replace without sed escape hell. Eliminates an entire error class per run. |
| **choose** | Column extraction with shorter syntax than `awk '{print $1,$3}'`. Negative indices, ranges. |

## Documents & Metadata

| Tool | Agent benefit |
|------|--------------|
| **pdftotext** (poppler) | Extracts text from PDFs for further processing. Makes PDFs searchable/parseable without loading the entire file as binary. Suite also includes `pdfinfo`, `pdfimages`. |
| **exiftool** | Structured read/write of metadata in images/PDF/audio/video. Only practical CLI for EXIF/IPTC/XMP — without it, metadata is a black box. |

## HTTP, API & Networking

| Tool | Agent benefit |
|------|--------------|
| **curl** | Universal HTTP. Always available. |
| **xh** | Shorter syntax than curl, JSON as default. Fewer characters per request. |
| **doggo** | DNS responses as JSON via `--json` → straight to `jq` without parsing. |
| **nmap** | Only practical port scanner. Checks service availability before I attempt to connect. |
| **grpcurl** | Only practical way to hit gRPC services from shell. Without it: no access to gRPC backends. |
| **websocat** | Only practical CLI for WebSocket APIs. Complements curl/grpcurl/hurl where the protocol is WS. |
| **hurl** | Declarative HTTP test files (`.hurl`). Executable documentation of API flows — one command per sequence instead of N curl calls with intermediate parsing. |
| **mitmdump** (mitmproxy) | Scriptable HTTP/HTTPS proxy. Capture and modify traffic to/from running services without code changes — only practical way to see what a client actually sends. |
| **buf** | Protobuf linting and compilation. Necessary complement to grpcurl: validates `.proto` files and generates stubs without manual `protoc` syntax. |
| **oha** | HTTP load testing in one command with statistics. Quick single-endpoint validation without setup. |
| **k6** | Scripted load testing with scenario support (ramp-up, stages, thresholds). Covers multi-step flows and complex patterns that oha cannot. |
| **rclone** | Universal cloud storage (S3/GCS/Azure/Dropbox/etc.) under one CLI. `rclone lsjson` gives JSON output straight to jq — structured access to buckets without provider-specific CLIs. |

## Linting & Static Analysis

| Tool | Agent benefit |
|------|--------------|
| **shellcheck** | Static analysis for shell scripts. Catches errors *before* execution — critical when writing bash since I don't see outcomes interactively. Eliminates an entire class of re-runs. |
| **shfmt** | Formats shell scripts automatically. Complements shellcheck: check + format in one round, no iteration over style errors. |
| **hadolint** | Dockerfile linting with best practices. Produces secure Dockerfiles without me remembering all rules. |
| **actionlint** | GitHub Actions workflow validation. Catches syntax and logic errors in CI config *before* push — saves an entire CI round per error. |
| **act** | Run GitHub Actions workflows locally without pushing. Test the entire CI pipeline before commit — saves minutes/hours per iteration. Complements actionlint: lint catches syntax, act catches logic errors. |
| **pre-commit** | Run repo hooks locally with `pre-commit run -a` and reproduce CI checks before push. Removes an entire class of "CI failed after commit" iterations. |
| **semgrep** | Semantic pattern-based code analysis with community rules. `semgrep --config auto .` finds security and quality issues across all languages in one run. |
| **biome** | Lightning-fast lint + format for JS/TS/JSON/CSS in one binary. Tight feedback loop like `ruff`, but for frontend/fullstack projects. |
| **ruff** | Python lint + format in milliseconds. Replaces flake8/black/isort/pyupgrade in one binary. Best tight feedback loop for Python — run on every file change without waiting. |
| **pyright** | Static type checking for Python. Catches errors before execution where ruff can't reach. Complements ruff: ruff finds style errors, pyright finds type errors. |
| **golangci-lint** | Aggregated Go linter that runs dozens of linters in parallel. Same shell-fast feedback as ruff, but for Go. |
| **yamllint** | Dedicated YAML linter (indent, duplicate keys, trailing space). `yq` doesn't validate style — yamllint catches style errors before they become CI failures. |
| **sqlfluff** | SQL lint + format with dialect support (postgres, mysql, snowflake, bigquery, etc.). Only practical SQL linter with multi-dialect support. |
| **vale** | Prose linter for docs/markdown/comments. Run on documentation PRs for consistent style — catches things code linters never see. |

## Security Scanning

| Tool | Agent benefit |
|------|--------------|
| **trivy** | One command for vulnerability scanning of container images, filesystems, and git repos. `trivy repo .` gives complete SBOM + CVE report without setup. |
| **trufflehog** | Scans git history for leaked secrets. `trufflehog git file://.` — one command, no interaction. Without it: blind to what may have been in history. |
| **gitleaks** | Fast secret scanning of working tree, commits, and history. Very well suited for pre-commit/CI where low runtime matters. |

## Kubernetes & Infrastructure

| Tool | Agent benefit |
|------|--------------|
| **kubectl** | Standard k8s CLI. `get`, `describe`, `logs`, `exec` — all data I need from the cluster without interactive TUI. |
| **helm** | `helm list`, `helm status`, `helm get values` gives state at release level. Indispensable for understanding what is deployed. |
| **kustomize** | `kustomize build` renders manifests to flat YAML — lets me inspect final configuration without rolling out. |
| **stern** | Aggregates logs from multiple pods with one pattern. Saves N separate `kubectl logs` calls when debugging multi-pod services. |
| **helmfile** | Declarative multi-chart Helm. `helmfile diff` shows what would change — perfect for pre-flight check without deploying. |
| **argocd** | ArgoCD CLI for GitOps. `argocd app get`, `argocd app sync`, `argocd app diff` — controls and inspects deployments via GitOps without dashboard. |
| **kubeconform** | Validates k8s manifests against OpenAPI schemas *offline*. Catches invalid YAML before apply — like hadolint/actionlint for k8s. |
| **conftest** | OPA/Rego policy against k8s/Terraform/Dockerfile manifests. Takes over where kubeconform/hadolint stop — enforces organizational rules (no `latest` tag, required resources, etc.) without writing a custom parser. |

## Build, Watch & Bench

| Tool | Agent benefit |
|------|--------------|
| **just** | Project commands in `justfile`. Discoverable: `just --list` shows what exists. |

## Secrets & Encryption

| Tool | Agent benefit |
|------|--------------|
| **gpg** | Signed commits, encryption. |

## Runtimes

| Tool | Agent benefit |
|------|--------------|
| **node** | JS/TS scripting and tooling. |
| **python3** | Scripting where shell doesn't suffice. |
| **go** | For building Go-based tools from source when brew version is missing. |
| **javac** | Java compiler |
| **java** | Java launcher |

## Unix Core Tools

`grep`, `sed`, `awk`, `find`, `xargs`, `cut`, `tr`, `wc`, `zip`, `unzip`, `xz`, `zstd`, `rsync`, `make` — listed only for completeness. Used where specialized tools aren't installed, or (for `make`) when the project already has a Makefile as entry point.
