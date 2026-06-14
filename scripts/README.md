# scripts

## `run-reference.sh`

Runs the compliance suite end-to-end against the **go-odata reference
implementation** ([`NLstn/go-odata`](https://github.com/NLstn/go-odata),
`cmd/complianceserver`). It clones (or reuses) the reference repo, builds and
starts its compliance server, runs this suite against it, and tears the server
down — propagating the suite's exit code so it can gate CI.

```bash
# Full run against a fresh clone of go-odata@main
./scripts/run-reference.sh

# Pass suite flags through
./scripts/run-reference.sh -verbose
./scripts/run-reference.sh -version 4.0 -pattern filter
```

### Requirements

- Go 1.25+ (the reference server's module requires it).
- A C compiler with `CGO_ENABLED=1` (the reference server uses
  `mattn/go-sqlite3`). `gcc` is enough.
- `git` on `PATH`.

### Environment overrides

| Variable        | Default                              | Purpose                                              |
|-----------------|--------------------------------------|------------------------------------------------------|
| `GO_ODATA_REPO` | `https://github.com/NLstn/go-odata`  | Reference implementation git URL.                    |
| `GO_ODATA_REF`  | `main`                               | Branch / tag / commit to check out.                  |
| `GO_ODATA_DIR`  | _(unset)_                            | Use an existing local checkout instead of cloning. Not deleted on exit. |
| `SERVER_PORT`   | `9090`                               | Port the reference server listens on.                |

Iterating locally? Clone go-odata once and point `GO_ODATA_DIR` at it to skip
re-cloning on every run:

```bash
git clone https://github.com/NLstn/go-odata /tmp/go-odata
GO_ODATA_DIR=/tmp/go-odata ./scripts/run-reference.sh -verbose
```

This is the same flow the `Reference compliance run` GitHub Actions workflow
(`.github/workflows/reference.yml`) uses.
