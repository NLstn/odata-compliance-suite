# OData v4 Compliance Test Suite

A standalone, black-box compliance test suite for **OData v4.0 and v4.01**
services. Point it at any running OData service and it reports how well that
service conforms to the specification.

The suite is written in Go but is **language-agnostic about the service under
test** — it only speaks HTTP. A .NET, Java, Python, Node, or Go OData service
can all be measured the same way.

> Extracted from the [`go-odata`](https://github.com/NLstn/go-odata) project,
> where it began as that library's internal conformance suite.

## What it checks

~106 suites / ~669 individual tests across:

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

## Requirements

- Go 1.24+ (to build/run the suite)
- A running OData service that exposes the **reference data model**
  documented in [`CONTRACT.md`](./CONTRACT.md).

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
```

### Exit codes

- `0` — all suites passed
- `1` — one or more tests failed, or the server was unreachable

Suitable for CI gating:

```yaml
- name: OData compliance
  run: |
    ./start-my-odata-service &   # your service on :8080
    go run . -server http://localhost:8080
```

## Output

**Normal mode** prints a single live progress line and a final summary:

```
Running 106 suites (669 total tests)
Progress: suites 106/106 | tests 669/669 | passed 669 | failed 0 | skipped 0

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
