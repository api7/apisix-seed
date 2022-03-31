ENV_INSTALL            ?= install
ENV_RM                 ?= rm -vf
ENV_INST_PREFIX        ?= /usr/local

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
	$(ENV_INSTALL) -d $(ENV_INST_PREFIX)/apisix-seed
	$(ENV_INSTALL) -d $(ENV_INST_PREFIX)/apisix-seed/log
	$(ENV_INSTALL) -d $(ENV_INST_PREFIX)/apisix-seed/conf
	$(ENV_INSTALL) apisix-seed $(ENV_INST_PREFIX)/apisix-seed/
	$(ENV_INSTALL) conf/conf.yaml $(ENV_INST_PREFIX)/apisix-seed/conf/

### uninstall:		Uninstall apisix-seed
.PHONY: uninstall
uninstall:
	$(ENV_RM) -r $(ENV_INST_PREFIX)/apisix-seed

### clean:		Clean apisix-seed
.PHONY: clean
clean:
	rm -f apisix-seed

