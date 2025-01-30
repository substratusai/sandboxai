BOX_IMG := substratusai/sandboxai-box:$(shell git describe --tags --dirty --always)

.PHONY: build-box-image
build-box-image:
	docker build . -f box.Dockerfile --progress=plain -t $(BOX_IMG)

UV := $(shell which uv)

install-uv:
ifndef UV
	curl -LsSf https://astral.sh/uv/install.sh | sh
else
	@echo "uv is already installed at $(UV)"
endif

.PHONY: build-sandboxaid
build-sandboxaid:
	cd go && go build -o ../bin/sandboxaid ./sandboxaid/main.go
	mkdir -p python/sandboxai/bin/
	cp bin/sandboxaid python/sandboxai/bin/

.PHONY: test-unit
test-unit:
	cd go && go test -v ./api/...
	cd go && go test -v ./client/...
	cd go && go test -v ./sandboxaid/...

.PHONY: test-e2e
test-e2e: install-uv build-sandboxaid build-box-image
	BOX_IMAGE=$(BOX_IMG) ./test/e2e/run.sh

.PHONY: lint-python
lint-python:
	cd python && uv run ruff check

.PHONY: format-python
format-python:
	cd python && uv run ruff format

.PHONY: format-python-check
format-python-check:
	cd python && uv run ruff format --check \
	   || (echo "Please run 'uv run ruff format' to fix this formatting issue." && exit 1)

.PHONY: test-all
test-all: test-unit test-e2e lint-python format-python-check

.PHONY: generate
generate: generate-go generate-python

.PHONY: generate-go
generate-go:
	# NOTE(nstogner): I prefer using "oapi-codegen/oapi-codegen"
	# over "openapitools/openapi-generator-cli" because it produces
	# cleaner Go code.
	cd ./go && go generate ./...

# TODO: Register the datamodel-code-generator pip package with UV.
.PHONY: generate-python
generate-python:
	datamodel-codegen \
		--input ./api/v1.yaml \
		--input-file-type openapi \
		--output ./python/sandboxai/api/v1.py

