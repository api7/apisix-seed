default: help

.PHONY: lint
lint:
	golangci-lint run --verbose ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: help
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'
