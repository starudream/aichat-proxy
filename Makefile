PROJECT := aichat-proxy
MODULE  := $(shell go list -m)
VERSION ?= $(shell git describe --exact-match --tags HEAD 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || date '+%Y%m%d-%H%M%S')

.PHONY: tidy
tidy:					##@ Tidy go.mod and go.sum.
	@go mod tidy

.PHONY: test
test: tidy				##@ Test all packages. (Recommended: https://github.com/gotestyourself/gotestsum)
	@if command -v gotestsum >/dev/null 2>&1; then \
    	gotestsum --format pkgname --format-icons text -- -race -count 1 -failfast -v ./...; \
	else \
		go test -race -count 1 -failfast -v ./...; \
	fi

.PHONY: lint
lint: tidy				##@ Lint all packages.
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout 5m; \
		echo "done."; \
	else \
		echo "golangci-lint is not installed. Please install it from https://github.com/golangci/golangci-lint"; \
		exit 1; \
	fi

.PHONY: bin
bin: tidy				##@ Build the application.
	@CGO_ENABLED=0 go build -tags 'release' -ldflags '-s -w' -o ./bin/$(PROJECT) $(MODULE)/server

.PHONY: bin-linux
bin-linux: tidy			##@ Build the application for linux.
	@GOOS=linux CGO_ENABLED=0 go build -tags 'release' -ldflags '-s -w' -o ./bin/$(PROJECT)-linux $(MODULE)/server

.PHONY: run
run: bin				##@ Run the application.
	@./bin/$(PROJECT) $(ARGS)

.PHONY: swag
swag:					##@ Generate swagger files.
	@go run github.com/swaggo/swag/cmd/swag@latest fmt -d server -g router/routes.go
	@go run github.com/swaggo/swag/cmd/swag@latest init -o server/docs -ot go,yaml -d server -g router/routes.go

.PHONY: help
help:					##@ (Default) Show help.
	@printf "\nUsage: make <command>\n"
	@grep -F -h "##@" $(MAKEFILE_LIST) | grep -F -v grep -F | sed -e 's/\\$$//' | awk 'BEGIN {FS = ":*[[:space:]]*##@[[:space:]]*"}; \
	{ \
		if($$2 == "") \
			pass; \
		else if($$0 ~ /^#/) \
			printf "\n%s\n", $$2; \
		else if($$1 == "") \
			printf "     %-20s%s\n", "", $$2; \
		else \
			printf "\n    \033[34m%-20s\033[0m %s\n", $$1, $$2; \
	}'
	@printf "\n"

.DEFAULT_GOAL := help
