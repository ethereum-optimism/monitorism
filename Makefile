COMPOSEFLAGS=-d
OP_STACK_GO_BUILDER?=us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest

build: build-go
.PHONY: build

build-go: op-monitorism op-defender
.PHONY: build-go

golang-docker:
	# We don't use a buildx builder here, and just load directly into regular docker, for convenience.
	GIT_COMMIT=$$(git rev-parse HEAD) \
	GIT_DATE=$$(git show -s --format='%ct') \
	IMAGE_TAGS=$$(git rev-parse HEAD),latest \
	docker buildx bake \
			--progress plain \
			--load \
			-f docker-bake.hcl \
			op-monitorism op-defender
.PHONY: golang-docker

op-monitorism:
	make -C ./op-monitorism 
.PHONY: op-monitorism

op-defender:
	make -C ./op-defender 
.PHONY: op-defender

lint-go: ## Lints Go code with specific linters
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...
.PHONY: lint-go

lint-go-fix: ## Lints Go code with specific linters and fixes reported issues
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./... --fix
.PHONY: lint-go-fix

tidy:
	make -C ./op-monitorism tidy
	make -C ./op-defender tidy
.PHONY: tidy

clean:
	rm -rf ./bin
.PHONY: clean

nuke: clean devnet-clean
	git clean -Xdf
.PHONY: nuke
