FROM golang:1.18-buster AS build-setup

RUN apt-get update
RUN apt-get -y install cmake zip sudo git

ENV FLOW_GO_REPO="https://github.com/onflow/flow-go"
ENV FLOW_GO_BRANCH=v0.26.16

RUN mkdir /dps /docker /flow-go

WORKDIR /dps

# clone repos
ADD . /dps
RUN git clone --branch $FLOW_GO_BRANCH $FLOW_GO_REPO /flow-go

RUN ln -s /flow-go /dps/flow-go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build  \
    make -C /flow-go crypto_setup_gopath #prebuild crypto dependency

RUN ls -la /flow-go/crypto/

FROM build-setup AS build-binary

ARG BINARY

WORKDIR /dps
RUN  --mount=type=cache,target=/go/pkg/mod \
     --mount=type=cache,target=/root/.cache/go-build  \
     GO111MODULE=on CGO_ENABLED=1 GOOS=linux go build -o /app --tags relic -ldflags "-extldflags -static" ./cmd/$BINARY && \
     chmod a+x /app


## Add the statically linked binary to a distroless image
FROM gcr.io/distroless/base-debian11  AS production

ARG BINARY

COPY --from=build-binary /app /app

EXPOSE 5005

ENTRYPOINT ["/app"]