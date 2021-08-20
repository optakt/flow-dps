export CONTAINER_REGISTRY := gcr.io/flow-container-registry

.PHONY: docker-build-create-index-snapshot
docker-build-create-index-snapshot:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=create-index-snapshot   \
		-t "$(CONTAINER_REGISTRY)/dps-create-index-snapshot:latest" .


.PHONY: docker-build-flow-access-server
docker-build-flow-access-server:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=flow-access-server   \
		-t "$(CONTAINER_REGISTRY)/dps-flow-access-server:latest" .


.PHONY: docker-build-flow-dps-client
docker-build-flow-dps-client:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=flow-dps-client  \
		-t "$(CONTAINER_REGISTRY)/dps-client:latest" .


.PHONY: docker-build-flow-dps-indexer
docker-build-flow-dps-indexer:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=flow-dps-indexer  \
		-t "$(CONTAINER_REGISTRY)/dps-indexer:latest" .


.PHONY: docker-build-flow-dps-server
docker-build-flow-dps-server:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=flow-dps-server  \
		-t "$(CONTAINER_REGISTRY)/dps-server:latest" .


.PHONY: docker-build-flow-rosetta-server
docker-build-flow-rosetta-server:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=flow-rosetta-server  \
		-t "$(CONTAINER_REGISTRY)/rosetta-server:latest" .


.PHONY: docker-build-restore-index-snapshot
docker-build-restore-index-snapshot:
	docker build -f cmd/Dockerfile  --build-arg COMMAND=restore-index-snapshot   \
		-t "$(CONTAINER_REGISTRY)/dps-restore-snapshot:latest" .


docker-build-all: docker-build-create-index-snapshot docker-build-flow-access-server docker-build-flow-dps-client docker-build-flow-dps-indexer docker-build-flow-dps-server docker-build-flow-rosetta-server docker-build-restore-index-snapshot
