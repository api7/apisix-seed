default: help

### help:		Show Makefile rules
.PHONY: help
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'

### go-lint:		Lint Go source codes
.PHONY: lint
lint:
	golangci-lint run --verbose ./...

### test:		Run the tests of apisix-seed
.PHONY: test
test:
	ENV=test go test -race -cover -coverprofile=coverage.txt ./...

### build:		Build apisix-seed
.PHONY: build
build:
	go build

### install:		Install apisix-seed
.PHONY: install
install:
	install -d /usr/local/apisix-seed
	install -d /usr/local/apisix-seed/log
	install -d /usr/local/apisix-seed/conf
	install apisix-seed /usr/local/apisix-seed/
	install conf/conf.yaml /usr/local/apisix-seed/conf/

### uninstall:		Uninstall apisix-seed
.PHONY: uninstall
uninstall:
	rm -rf /usr/local/apisix-seed
