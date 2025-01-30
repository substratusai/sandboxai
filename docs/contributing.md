# Contributing

## Run from Source

Build components from source.

```bash
make build-sandboxaid build-box-image
```

Run an example.

```bash
cd python && uv run ./examples/basic-with-logging.py
```

## Run Tests

Run tests with make:

```bash
make test-unit
make test-e2e
```
