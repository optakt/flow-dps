FROM golang:1.19-buster AS build-setup

RUN apt-get update \
    && apt-get -y install cmake zip sudo git

RUN mkdir /archive
WORKDIR /archive
COPY . /archive
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build  \
    bash crypto_build.sh

FROM build-setup AS build-binary

ARG BINARY

WORKDIR /archive
RUN	--mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build  \
    CGO_ENABLED=1 GOOS=linux go build -o /app --tags "relic,netgo" -ldflags "-extldflags -static" ./cmd/$BINARY && \
    chmod a+x /app

## Add the statically linked binary to a distroless image
FROM gcr.io/distroless/base-debian11  AS production

ARG BINARY

COPY --from=build-binary /app /app

EXPOSE 5005

ENTRYPOINT ["/app"]