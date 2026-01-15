# Contributing

## Setup

```bash
go mod download
```

## Development

```bash
make test  # Run unit tests
make lint  # Lint and fix violations
make build # Build to bin/grove
```

## Pull requests

1. Branch from `main`
2. Add a changelog entry via `make change`
3. Verify `make ci` passes
4. Open a PR to `main`
