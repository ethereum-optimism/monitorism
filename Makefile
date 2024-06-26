COMPOSEFLAGS=-d
OP_STACK_GO_BUILDER?=us-docker.pkg.dev/oplabs-tools-artifacts/images/op-stack-go:latest

build: build-go
.PHONY: build

build-go: op-monitorism
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
			op-monitorism
.PHONY: golang-docker

op-monitorism:
	make -C ./op-monitorism 
.PHONY: op-monitorism

tidy:
	make -C ./op-monitorism tidy
.PHONY: tidy

clean:
	rm -rf ./bin
.PHONY: clean

nuke: clean devnet-clean
	git clean -Xdf
.PHONY: nuke