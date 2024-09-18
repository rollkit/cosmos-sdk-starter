# Define pkgs, run, and cover variables for test so that we can override them in
# the terminal more easily.
pkgs := $(shell go list ./...)
run := .
count := 1

## help: Show this help message
help: Makefile
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
.PHONY: help

## cover: generate to code coverage report.
# cover:
# 	@echo "--> Generating Code Coverage"
# 	@go install github.com/ory/go-acc@latest
# 	@go-acc -o coverage.txt $(pkgs)
# .PHONY: cover

## deps: Install dependencies
deps:
	@echo "--> Installing dependencies"
	@go mod download
	@go mod tidy
.PHONY: deps

## lint: Run linters golangci-lint and markdownlint.
lint: vet
	@echo "--> Running golangci-lint"
	@golangci-lint run
	@echo "--> Running markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md'
	@echo "--> Running yamllint"
	@yamllint --no-warnings . -c .yamllint.yml

.PHONY: lint

## fmt: Run fixes for linters.
fmt:
	@echo "--> Formatting markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md' -f
	@echo "--> Formatting go"
	@golangci-lint run --fix
.PHONY: fmt

## vet: Run go vet
vet: 
	@echo "--> Running go vet"
	@echo $(pkgs)
	@go vet $(pkgs)
.PHONY: vet

## test: Running unit tests
test: vet
	@echo "--> No unit tests"
# 	@go test -v -race -covermode=atomic -coverprofile=coverage.txt $(pkgs) -run $(run) -count=$(count)
.PHONY: test
