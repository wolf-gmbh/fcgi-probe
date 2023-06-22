SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ Options

releaser ?= go run github.com/goreleaser/goreleaser
linter ?= go run github.com/golangci/golangci-lint/cmd/golangci-lint


##@ Commands

help: ## Display this help text
	@./Makehelp Makefile


###@ Development

vet: ## run static code checks
	go vet ./...
	$(linter) run

test: ## run tests
	go test ./...

build: ## build the binaries for all targets
	$(releaser) build --clean --snapshot


###@ Release

tag: ## create a new git tag based on conventional commit messages
	@$(eval git_tag=v$(shell docker run -v "$(CURDIR):/tmp" --workdir /tmp \
		--rm -u '$(shell id -u):$(shell id -g)' convco/convco version --bump))
	@git tag -m "bump" -f "$(git_tag)"

release: tag ## create a new tag and release
	$(releaser) release --clean

release-dryrun: tag ## same as release but local-only
	$(releaser) --clean --skip-publish


###@ Pipeline

pipeline-validation: test vet build ## validatate the project

pipeline-release: release ## release the project
