# OData v4 Compliance Test Suite

A standalone, black-box compliance test suite for **OData v4.0 and v4.01**
services. Point it at any running OData service and it reports which
specification checks pass, fail, or are not exercised.

The suite is written in Go but is **language-agnostic about the service under
test** — it only speaks HTTP. A .NET, Java, Python, Node, or Go OData service
can all be measured the same way.

> Extracted from the [`go-odata`](https://github.com/NLstn/go-odata) project,
> where it began as that library's internal conformance suite.

## What it checks

162 suites / 1,188 individual tests across:

- Service document & metadata (`$metadata` XML + JSON)
- Query options: `$filter`, `$select`, `$orderby`, `$top`, `$skip`, `$expand`,
  `$count`, `$search`, `$compute`, `$apply` (full transformation catalog),
  `$format`, `$index`, `$skiptoken`, parameter aliases
- CRUD, upsert, deep insert, batch (`multipart/mixed` and JSON batch)
- Relationships, navigation, `$ref`, type casting, derived types
- HTTP headers, content negotiation, conditional requests, ETags
- Error response shape and consistency
- Functions & actions (incl. overloading), async processing
- Vocabulary annotations: Core, Capabilities
- OData 4.01-specific features (`in` operator, `divby`, key-as-segments,
  JSON batch, wildcard `$select`/`$expand`, `matchesPattern`, …)

The Minimal/Intermediate/Advanced summary is a suite coverage grouping, not a
formal OASIS certification. See [`CONFORMANCE.md`](./CONFORMANCE.md); skipped
tests make a band incomplete rather than silently counting it as met.

## Requirements

- A running OData service that exposes the **reference data model**
  documented in [`CONTRACT.md`](./CONTRACT.md).
- Go 1.24+ *only* if you build from source or use `go run`. The
  [prebuilt binary](#3-prebuilt-binary), [Docker image](#2-docker-image), and
  [GitHub Action](#1-github-action-ci) need no Go toolchain.

> **Important:** v1 requires the exact reference model (entity sets like
> `Products`/`Categories`, the `Company` singleton, specific seed rows, and a
> set of bound/unbound operations). Read [`CONTRACT.md`](./CONTRACT.md) before
> running against your own service. Support for configurable / discovered
> models is on the roadmap.

## Usage

```bash
# Run all suites against a service already running at the default URL
go run . -server http://localhost:9090

# Or build a binary
go build -o compliance-test .
./compliance-test -server http://localhost:8080

# Only OData 4.0, or 4.01, or vocabulary suites
./compliance-test -version 4.0
./compliance-test -version 4.01
./compliance-test -version vocabularies

# Run a subset of suites by name substring
./compliance-test -pattern filter

# Verbose (per-test results) or debug (full HTTP) output
./compliance-test -verbose
./compliance-test -debug
```

The suite does **not** start a server — start your service first and point
`-server` at its root. It waits up to `-timeout` seconds for the root to
return HTTP 200 before running.

### Flags

```
-server   URL of the OData service under test (default http://localhost:9090)
-version  4.0 | 4.01 | vocabularies | all (default all)
-pattern  Run only suites whose name contains this substring
-verbose  Show every individual test result
-debug    Show full HTTP request/response details
-timeout  Seconds to wait for the server to become reachable (default 30)
-strict   Treat capability-skipped tests as failures (see below)
```

### Capability-aware skipping

Before running any tests the suite fetches `/$metadata` and reads the service's
`Org.OData.Capabilities.V1` annotations. When a suite requires a feature (e.g.
`$filter`, `$batch`, INSERT) and the service has declared that feature unsupported,
the suite is **skipped** instead of run-and-failed. Skipped suites are reported
separately and never counted as failures.

Supported annotations:

| Annotation | Capability gated |
|---|---|
| `FilterRestrictions.Filterable=false` on an entity set | all `$filter` suites |
| `SortRestrictions.Sortable=false` | `$orderby` suites |
| `ExpandRestrictions.Expandable=false` | `$expand` suites |
| `CountRestrictions.Countable=false` | `$count` suites |
| `SearchRestrictions.Searchable=false` | `$search` suites |
| `InsertRestrictions.Insertable=false` | create / deep-insert / upsert suites |
| `UpdateRestrictions.Updatable=false` | update / upsert suites |
| `DeleteRestrictions.Deletable=false` | delete suite |
| `TopSupported=false` | `$top` tests |
| `SkipSupported=false` (container) | skip/skiptoken/pagination suites |
| `BatchSupported=false` or `BatchSupport.Supported=false` | multipart and JSON batch suites |
| `ComputeSupported=false` | `$compute` suites |
| `SelectSupport.Supported=false` | `$select` tests |
| `KeyAsSegmentSupported=false` | key-as-segment suite |
| `ReadRestrictions.Readable=false` | entity-read tests |
| `IndexableByKey=false` | key-addressing tests |

The parser understands inline and external annotations, vocabulary/schema
aliases, attribute or element boolean expressions, and container-level
`DefaultCapabilities` defaults.

If `$metadata` cannot be fetched or parsed, the suite warns and runs all tests
(fail-open).

Use `-strict` to disable this behaviour — unsupported-capability suites run and
fail normally. This is useful when you expect a service to be fully conformant
and want to catch accidental capability declarations.

### Exit codes

- `0` — all suites passed
- `1` — one or more tests failed, or the server was unreachable

## Consuming the suite

The suite never starts a service — **you** start one that implements
[`CONTRACT.md`](./CONTRACT.md), then point the suite at its root URL. There are
four ways to run it; pick by your toolchain and where you run it.

| Channel | Best for |
|---|---|
| [GitHub Action](#1-github-action-ci) | CI for any service started as a host process (e.g. Go) |
| [Docker image](#2-docker-image) | Any language; local runs without a Go toolchain |
| [Prebuilt binary](#3-prebuilt-binary) | Local dev on any machine; CI without Docker |
| [`go run`](#4-go-run) | Go users; quick local runs |

In every case the suite exits `0` when compliant and `1` on failure (or if the
service is unreachable), so it gates CI directly.

### 1. GitHub Action (CI)

Start your service in the background, then run the Action against it:

```yaml
- name: Start OData service   # must expose the CONTRACT.md model
  run: ./start-my-odata-service &   # listening on :9090

- name: OData compliance
  uses: NLstn/odata-compliance-suite@v1
  with:
    server: http://localhost:9090
    # version: 4.01      # optional: 4.0 | 4.01 | vocabularies | all
    # pattern: filter    # optional: run only matching suites
    # verbose: 'true'    # optional
```

The Action downloads a prebuilt binary and runs it **on the runner host**, so a
service listening on `localhost` is reachable with no extra networking. Pin the
binary with `suite-version: v1.2.3` (defaults to the latest release). To test an
unreleased action ref, set `suite-version: source`; the Action then builds and
runs the suite from that exact ref using the runner's Go toolchain.

### 2. Docker image

Language-agnostic, no Go required:

```bash
docker run --rm ghcr.io/nlstn/odata-compliance-suite:v1 \
  -server http://my-service:9090 -version 4.01
```

> **Networking note:** inside the container, `localhost` is the *container*, not
> your host. To reach a service running on the host, use
> `--network host` (Linux) and `-server http://localhost:9090`, or
> `-server http://host.docker.internal:9090` (Docker Desktop). In CI, prefer
> running the service-under-test as a service container (or on the same Docker
> network) and addressing it by name.

### 3. Prebuilt binary

Download a static binary for your platform from the
[Releases](https://github.com/NLstn/odata-compliance-suite/releases) page:

```bash
curl -fsSL -o compliance-test \
  https://github.com/NLstn/odata-compliance-suite/releases/latest/download/odata-compliance-suite_linux_amd64
chmod +x compliance-test
./compliance-test -server http://localhost:9090
```

### 4. `go run`

For Go users, no checkout needed:

```bash
go run github.com/nlstn/odata-compliance-suite@latest -server http://localhost:9090
```

### Worked example: a Go service (`go-odata`)

A Go OData library can start its reference server and run the suite against it.
In CI:

```yaml
- name: Start OData service (exposes the CONTRACT.md model)
  run: go run ./cmd/complianceserver -db sqlite -port 9090 &
  # The suite waits up to -timeout for the root to return 200, and calls
  # POST /Reseed itself before each run.

- name: Run compliance suite
  uses: NLstn/odata-compliance-suite@v1
  with:
    server: http://localhost:9090
```

Locally it's the same two steps: start the server, then run the suite via any
channel above.

## Output

**Normal mode** prints a single live progress line and a final summary:

```
Running 162 suites (1188 total tests)
Progress: suites 162/162 | tests 1188/1188 | passed 1188 | failed 0 | skipped 0

Test Scripts: 106/106 passed (100%)
Individual Tests:
  - Total: 669
  - Passing: 669
  - Failing: 0
  - Skipped: 0
  - Pass Rate: 100%
```

**Verbose mode** prints each test (`✓ PASS`, `✗ FAIL`, `⊘ SKIP`) with error
detail for failures.

## Project structure

```
.
├── main.go            # CLI runner: registers suites, drives the run, reports
├── framework/         # HTTP client + assertion helpers (pure stdlib)
├── tests/
│   ├── v4_0/          # OData 4.0 protocol suites
│   ├── v4_01/         # OData 4.01 protocol suites
│   └── vocabularies/  # Core & Capabilities vocabulary suites
├── CONTRACT.md        # Reference data-model the service must expose
├── go.mod             # module github.com/nlstn/odata-compliance-suite (no deps)
└── LICENSE
```

## Writing a new test

Add a function returning a `*framework.TestSuite` in the appropriate `tests/`
package, then register it in `main.go`.

```go
package v4_0

import "github.com/nlstn/odata-compliance-suite/framework"

func MyNewSuite() *framework.TestSuite {
    suite := framework.NewTestSuite(
        "Section Name",
        "What this validates",
        "https://link-to-odata-spec-section",
    )

    suite.AddTest("test_name", "Human-readable description",
        func(ctx *framework.TestContext) error {
            resp, err := ctx.GET("/Products")
            if err != nil {
                return err
            }
            if err := ctx.AssertStatusCode(resp, 200); err != nil {
                return err
            }
            return ctx.AssertJSONField(resp, "@odata.context")
        })

    return suite
}
```

Register it in `main.go`:

```go
testSuites = append(testSuites, TestSuiteInfo{
    Name:    "my_new_suite",
    Version: "4.0",
    Suite:   v4_0.MyNewSuite,
})
```

### Assertion helpers (`TestContext`)

```go
ctx.GET(path) / ctx.POST(path, body, headers...) / ctx.PATCH / ctx.PUT / ctx.DELETE
ctx.AssertStatusCode(resp, 200)
ctx.AssertHeader(resp, "Content-Type", "application/json")
ctx.AssertHeaderContains(resp, "Content-Type", "json")
ctx.AssertJSONField(resp, "@odata.context")
ctx.GetJSON(resp, &target)
ctx.IsValidJSON(resp)
ctx.AssertBodyContains(resp, "expected text")
ctx.Skip("reason")
```

## Reference server

The reference implementation of [`CONTRACT.md`](./CONTRACT.md) lives in the
[`go-odata`](https://github.com/NLstn/go-odata) repo under
`cmd/complianceserver`. To validate the suite itself, run that server and point
`-server` at it.

## License

MIT — see [`LICENSE`](./LICENSE).
