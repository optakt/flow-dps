# The tag of the current commit, otherwise empty
VERSION := $(shell git describe --tags --abbrev=2 --match "v*")

# The short Git commit hash
SHORT_COMMIT := $(shell git rev-parse --short HEAD)

# Image tag: if image tag is not set, set it with version (or short commit if empty)

ifeq (${IMAGE_TAG},)
IMAGE_TAG := ${VERSION}
endif

export DOCKER_BUILDKIT := 1

ALL_PACKAGES := ./...

# docker container registry
export CONTAINER_REGISTRY := gcr.io/flow-container-registry

# Dev Utilities
#############################################################################################################

.PHONY: tidy
tidy:
	go mod tidy -v
	git diff --exit-code

.PHONY: lint
lint: tidy
	golangci-lint run -v --build-tags relic ./...

.PHONY: fix-lint
fix-lint:
	golangci-lint run -v --build-tags relic --fix ./...

.PHONY: generate
generate: buf-generate

.PHONY: unittest
unittest:
	go test -tags relic -v $(ALL_PACKAGES)

.PHONY: compile
compile:
	go build -tags relic $(ALL_PACKAGES)

.PHONY: integ-test
integ-test:
	go test -v -tags="relic integration" $(ALL_PACKAGES)

.PHONY: test
test: unittest integ-test

.PHONY: buf-generate
buf-generate:
	cd api/protobuf && buf generate .

.PHONY: crypto_setup
crypto_setup:
	bash crypto_build.sh

# Docker Utilities! Do not delete these targets
#############################################################################################################

.PHONY: docker-build-live
docker-build-live:
	 docker build --build-arg BINARY=flow-archive-live . -t "$(CONTAINER_REGISTRY)/flow-archive-live:$(IMAGE_TAG)"

.PHONY: docker-build-indexer
docker-build-indexer:
	docker build --build-arg BINARY=flow-archive-indexer . -t "$(CONTAINER_REGISTRY)/flow-archive-indexer:$(IMAGE_TAG)"

.PHONY: docker-build-client
docker-build-client:
	docker build --build-arg BINARY=flow-archive-client . -t "$(CONTAINER_REGISTRY)/flow-archive-client:$(IMAGE_TAG)"

.PHONY: docker-build-server
docker-build-server:
	docker build --build-arg BINARY=flow-archive-server . -t "$(CONTAINER_REGISTRY)/flow-archive-server:$(IMAGE_TAG)"

.PHONY: docker-build-flow-archive
docker-build-flow-archive: docker-build-live docker-build-indexer docker-build-client docker-build-server

.PHONY: docker-push-live
docker-push-live:
	docker push "$(CONTAINER_REGISTRY)/flow-archive-live:$(IMAGE_TAG)"

.PHONY: docker-push-indexer
docker-push-indexer:
	docker push "$(CONTAINER_REGISTRY)/flow-archive-indexer:$(IMAGE_TAG)"

.PHONY: docker-push-client
docker-push-client:
	docker push "$(CONTAINER_REGISTRY)/flow-archive-client:$(IMAGE_TAG)"

.PHONY: docker-push-server
docker-push-server:
	docker push "$(CONTAINER_REGISTRY)/flow-archive-server:$(IMAGE_TAG)"

.PHONY: docker-push-flow-archive
docker-push-flow-archive: docker-push-live docker-push-indexer docker-push-client docker-push-server


PHONY: docker-build-create-checkpoint
docker-build-create-checkpoint:
	docker build -f cmd/Dockerfile --build-arg TARGET=./cmd/create-checkpoint --build-arg GOARCH=$(GOARCH) --target production \
		-t "$(CONTAINER_REGISTRY)/create-checkpoint:latest" -t "$(CONTAINER_REGISTRY)/create-checkpoint:$(SHORT_COMMIT)" -t "$(CONTAINER_REGISTRY)/create-checkpoint:$(IMAGE_TAG)" .

PHONY: tool-create-checkpoint
tool-create-checkpoint: docker-build-create-checkpoint
	docker container create --name create-checkpoint $(CONTAINER_REGISTRY)/create-checkpoint:latest;docker container cp create-checkpoint:/bin/app ./create-checkpoint;docker container rm create-checkpoint
