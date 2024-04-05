.PHONY: deps test unit-test

REPO_PATH := github.com/yuyang0/dagflow
REVISION := $(shell git rev-parse HEAD || unknown)
BUILTAT := $(shell date +%Y-%m-%dT%H:%M:%S)
VERSION := $(shell git describe --tags $(shell git rev-list --tags --max-count=1))
GO_LDFLAGS ?= -X $(REPO_PATH)/version.REVISION=$(REVISION) \
			  -X $(REPO_PATH)/version.BUILTAT=$(BUILTAT) \
			  -X $(REPO_PATH)/version.VERSION=$(VERSION)
deps:
	go mod vendor


test: deps unit-test

unit-test:
	go vet `go list ./... | grep -v '/vendor/' | grep -v '/tools'` && \
	go test -race -timeout 600s -count=1 -vet=off -cover ./utils/... \
	./flow/... \
	./service/... 

lint:
	golangci-lint run
