# The tag of the current commit, otherwise empty
VERSION := $(shell git describe --tags --abbrev=2 --match "v*")

# The short Git commit hash
SHORT_COMMIT := $(shell git rev-parse --short HEAD)

# Image tag: if image tag is not set, set it with version (or short commit if empty)
ifeq (${IMAGE_TAG},)
IMAGE_TAG := ${VERSION}
endif

ifeq (${IMAGE_TAG},)
IMAGE_TAG := ${SHORT_COMMIT}
endif

ALL_PACKAGES := ./...

# docker container registry
export CONTAINER_REGISTRY := gcr.io/flow-container-registry

# Dev Utilities
#############################################################################################################

.PHONY: generate
generate:
	go generate ./...

.PHONY: unittest
unittest:
	go test -tags relic -v ./...

.PHONY: compile
compile:
	go build -tags relic ./...

.PHONY: integ-test
integ-test:
	go test -v -tags="relic integration" ./...

.PHONY: test
test: unittest integ-test

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
docker-build: docker-build-live docker-build-indexer docker-build-client docker-build-server

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
docker-push: docker-push-live docker-push-indexer docker-push-client docker-push-server
